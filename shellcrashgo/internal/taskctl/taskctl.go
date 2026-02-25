package taskctl

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"shellcrash/internal/coreconfig"
	"shellcrash/internal/initctl"
	"shellcrash/internal/lifecycle"
	"shellcrash/internal/startctl"
)

var (
	idPattern      = regexp.MustCompile(`^[1-9][0-9][0-9]$`)
	errTaskUnknown = errors.New("unknown builtin task")
	coreConfigRun  = coreconfig.Run
	startActionRun = func(crashDir, action string) error {
		return startActionRunWithArgs(crashDir, action, nil)
	}
	startActionRunWithArgs = func(crashDir, action string, extraArgs []string) error {
		cfg, err := startctl.LoadConfig(crashDir)
		if err != nil {
			return err
		}
		ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
		return ctl.RunWithArgs(action, "", false, extraArgs)
	}
	webSaveRun = lifecycle.SaveWebSelections
	initctlRun = initctl.Run
	taskHTTPDo = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{Timeout: 20 * time.Second}
		return client.Do(req)
	}
	runTaskCommand = func(cfg startctl.Config, name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Env = append(os.Environ(),
			"CRASHDIR="+cfg.CrashDir,
			"TMPDIR="+cfg.TmpDir,
			"BINDIR="+cfg.BinDir,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
)

type Runner struct {
	CrashDir string
}

func (r Runner) Run(args []string) error {
	if r.CrashDir == "" {
		r.CrashDir = "/etc/ShellCrash"
	}
	if len(args) == 0 {
		return RunMenu(MenuOptions{CrashDir: r.CrashDir, In: os.Stdin, Out: os.Stdout, Err: os.Stderr})
	}
	if strings.EqualFold(strings.TrimSpace(args[0]), "menu") {
		return RunMenu(MenuOptions{CrashDir: r.CrashDir, In: os.Stdin, Out: os.Stdout, Err: os.Stderr})
	}

	cfg, err := startctl.LoadConfig(r.CrashDir)
	if err != nil {
		return err
	}

	arg := args[0]
	if idPattern.MatchString(arg) {
		cmd, name, err := resolveTaskCommand(r.CrashDir, arg)
		if err != nil {
			return err
		}
		displayName := name
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			displayName = args[1]
		}
		err = runTaskListCommand(cfg, r.CrashDir, cmd)
		if err != nil {
			_ = writeTaskLog(cfg.TmpDir, fmt.Sprintf("任务【%s】执行失败", displayName))
			return err
		}
		_ = writeTaskLog(cfg.TmpDir, fmt.Sprintf("任务【%s】执行成功", displayName))
		return nil
	}

	if err := runBuiltinTask(cfg, r.CrashDir, arg); err == nil {
		return nil
	} else if !errors.Is(err, errTaskUnknown) {
		return err
	}
	if len(args) == 1 {
		return runTaskExpression(cfg, args[0])
	}
	return runTaskCommand(cfg, args[0], args[1:]...)
}

func runBuiltinTask(cfg startctl.Config, crashDir, arg string) error {
	switch arg {
	case "update_config":
		_, err := coreConfigRun(coreconfig.Options{CrashDir: crashDir, TmpDir: cfg.TmpDir})
		if err != nil {
			return err
		}
		return startActionRun(crashDir, "start")
	case "hotupdate":
		return runHotUpdate(cfg)
	case "reset_firewall":
		if err := startActionRun(crashDir, "stop_firewall"); err != nil {
			return err
		}
		return startActionRun(crashDir, "afstart")
	case "ntp":
		if _, err := exec.LookPath("ntpd"); err != nil {
			return nil
		}
		return exec.Command("ntpd", "-n", "-q", "-p", "203.107.6.88").Run()
	case "web_save_auto":
		return webSaveRun(cfg.CrashDir, cfg.TmpDir, cfg.DBPort, cfg.Secret)
	case "update_mmdb":
		return runUpdateMMDB(cfg)
	case "update_core":
		return runUpdateCore(cfg)
	case "update_scripts":
		return runUpdateScripts(cfg)
	default:
		return errTaskUnknown
	}
}

func runTaskListCommand(cfg startctl.Config, crashDir, command string) error {
	if wrapped, ok := parseWrappedScriptAction(command, crashDir, "start.sh"); ok {
		return startActionRunWithArgs(crashDir, wrapped.Action, wrapped.Args)
	}
	if wrapped, ok := parseWrappedScriptAction(command, crashDir, "task.sh"); ok {
		if len(wrapped.Args) != 0 {
			return runTaskExpression(cfg, command)
		}
		if err := runBuiltinTask(cfg, crashDir, wrapped.Action); err == nil {
			return nil
		} else if !errors.Is(err, errTaskUnknown) {
			return err
		}
		return fmt.Errorf("unsupported task action %q", wrapped.Action)
	}
	expanded := expandTaskVars(command, cfg)
	if pipeline, ok := parseDirectCommandPipeline(expanded); ok {
		return runDirectCommandPipeline(cfg, pipeline)
	}
	return fmt.Errorf("unsupported shell expression in task command: %s", strings.TrimSpace(command))
}

type taskCommand struct {
	Name string
	Args []string
}

type wrappedScriptAction struct {
	Action string
	Args   []string
}

func parseWrappedScriptAction(command, crashDir, scriptName string) (wrappedScriptAction, bool) {
	fields := strings.Fields(expandTaskVars(command, startctl.Config{CrashDir: crashDir}))
	if len(fields) < 2 {
		return wrappedScriptAction{}, false
	}
	scriptIdx := 0
	if isShellInterpreter(fields[0]) {
		if len(fields) < 3 {
			return wrappedScriptAction{}, false
		}
		scriptIdx = 1
	}
	if filepath.Base(fields[scriptIdx]) != scriptName {
		return wrappedScriptAction{}, false
	}
	if scriptIdx == 1 && len(fields) < 3 {
		return wrappedScriptAction{}, false
	}
	action := fields[scriptIdx+1]
	if strings.TrimSpace(action) == "" {
		return wrappedScriptAction{}, false
	}
	args := append([]string{}, fields[scriptIdx+2:]...)
	return wrappedScriptAction{
		Action: action,
		Args:   args,
	}, true
}

func isShellInterpreter(name string) bool {
	switch filepath.Base(name) {
	case "sh", "ash", "bash":
		return true
	default:
		return false
	}
}

func expandCrashDir(command, crashDir string) string {
	out := strings.TrimSpace(command)
	out = strings.ReplaceAll(out, "${CRASHDIR}", crashDir)
	out = strings.ReplaceAll(out, "$CRASHDIR", crashDir)
	return out
}

func expandTaskVars(command string, cfg startctl.Config) string {
	out := expandCrashDir(command, cfg.CrashDir)
	out = strings.ReplaceAll(out, "${TMPDIR}", cfg.TmpDir)
	out = strings.ReplaceAll(out, "$TMPDIR", cfg.TmpDir)
	out = strings.ReplaceAll(out, "${BINDIR}", cfg.BinDir)
	out = strings.ReplaceAll(out, "$BINDIR", cfg.BinDir)
	return out
}

func parseDirectCommandPipeline(command string) ([]taskCommand, bool) {
	text := strings.TrimSpace(command)
	if text == "" {
		return nil, false
	}
	if hasUnsupportedShellSyntax(text) {
		return nil, false
	}
	parts, ok := splitCommandByAndAnd(text)
	if !ok {
		return nil, false
	}
	out := make([]taskCommand, 0, len(parts))
	for _, part := range parts {
		fields, err := splitCommandFields(part)
		if err != nil || len(fields) == 0 {
			return nil, false
		}
		out = append(out, taskCommand{Name: fields[0], Args: fields[1:]})
	}
	return out, true
}

func runTaskExpression(cfg startctl.Config, expression string) error {
	expanded := expandTaskVars(expression, cfg)
	if pipeline, ok := parseDirectCommandPipeline(expanded); ok {
		return runDirectCommandPipeline(cfg, pipeline)
	}
	return fmt.Errorf("unsupported shell expression: %s", strings.TrimSpace(expression))
}

func hasUnsupportedShellSyntax(text string) bool {
	inSingle := false
	inDouble := false
	escaped := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case escaped:
			escaped = false
			continue
		case r == '\\' && !inSingle:
			escaped = true
			continue
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			continue
		case r == '"' && !inSingle:
			inDouble = !inDouble
			continue
		}
		if inSingle || inDouble {
			continue
		}
		switch r {
		case ';', '`', '(', ')', '<', '>':
			return true
		case '|':
			return true
		case '&':
			if i+1 >= len(runes) || runes[i+1] != '&' {
				return true
			}
			i++
		}
	}
	return false
}

func splitCommandByAndAnd(text string) ([]string, bool) {
	parts := []string{}
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
			continue
		case r == '\\' && !inSingle:
			cur.WriteRune(r)
			escaped = true
			continue
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			cur.WriteRune(r)
			continue
		case r == '"' && !inSingle:
			inDouble = !inDouble
			cur.WriteRune(r)
			continue
		}
		if !inSingle && !inDouble && r == '&' {
			if i+1 >= len(runes) || runes[i+1] != '&' {
				return nil, false
			}
			part := strings.TrimSpace(cur.String())
			if part == "" {
				return nil, false
			}
			parts = append(parts, part)
			cur.Reset()
			i++
			continue
		}
		cur.WriteRune(r)
	}
	if escaped || inSingle || inDouble {
		return nil, false
	}
	last := strings.TrimSpace(cur.String())
	if last == "" {
		return nil, false
	}
	parts = append(parts, last)
	return parts, true
}

func splitCommandFields(text string) ([]string, error) {
	args := make([]string, 0, 8)
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		args = append(args, cur.String())
		cur.Reset()
	}
	for _, r := range text {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inSingle || inDouble {
				cur.WriteRune(r)
				continue
			}
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("invalid command quoting")
	}
	flush()
	return args, nil
}

func runDirectCommandPipeline(cfg startctl.Config, pipeline []taskCommand) error {
	for _, c := range pipeline {
		if err := runTaskCommand(cfg, c.Name, c.Args...); err != nil {
			return err
		}
	}
	return nil
}

func runHotUpdate(cfg startctl.Config) error {
	res, err := coreConfigRun(coreconfig.Options{CrashDir: cfg.CrashDir, TmpDir: cfg.TmpDir})
	if err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(cfg.TmpDir, "CrashCore"))

	format := res.Format
	if format == "" {
		format = "yaml"
	}
	path := res.CoreConfig
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(cfg.CrashDir, format+"s", "config."+format)
	}
	return hotReloadConfig(cfg.DBPort, cfg.Secret, path)
}

func hotReloadConfig(dbPort, secret, configPath string) error {
	if strings.TrimSpace(dbPort) == "" || strings.TrimSpace(configPath) == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"path": configPath})
	req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:"+dbPort+"/configs", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(secret) != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := taskHTTPDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("hotupdate reload failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func runUpdateMMDB(cfg startctl.Config) error {
	cfgPath := filepath.Join(cfg.CrashDir, "configs", "ShellCrash.cfg")
	cfgKV, err := parseKV(cfgPath)
	if err != nil {
		return err
	}
	remoteVer, err := fetchRemoteVersionKV(cfg, cfgKV)
	if err != nil {
		return err
	}
	geoVer := strings.TrimSpace(remoteVer["GeoIP_v"])
	if geoVer == "" {
		return fmt.Errorf("remote GeoIP_v is empty")
	}

	type item struct {
		localFile string
		remote    string
		cfgKey    string
	}
	items := []item{
		{localFile: "Country.mmdb", remote: "cn_mini.mmdb", cfgKey: "cn_mini_v"},
		{localFile: "cn_ip.txt", remote: "china_ip_list.txt", cfgKey: "china_ip_list_v"},
		{localFile: "cn_ipv6.txt", remote: "china_ipv6_list.txt", cfgKey: "china_ipv6_list_v"},
		{localFile: "GeoSite.dat", remote: "geosite.dat", cfgKey: "geosite_v"},
		{localFile: "geoip.db", remote: "geoip_cn.db", cfgKey: "geoip_cn_v"},
		{localFile: "geosite.db", remote: "geosite_cn.db", cfgKey: "geosite_cn_v"},
	}

	changed := false
	for _, it := range items {
		if strings.TrimSpace(cfgKV[it.cfgKey]) == "" {
			continue
		}
		srcLocal := filepath.Join(cfg.CrashDir, it.localFile)
		if st, statErr := os.Stat(srcLocal); statErr != nil || st.Size() == 0 {
			continue
		}
		if strings.TrimSpace(cfgKV[it.cfgKey]) == geoVer {
			continue
		}

		rel := filepath.ToSlash(filepath.Join("bin", "geodata", it.remote))
		url, err := resolveProjectAssetURL(cfg.CrashDir, cfgKV, rel)
		if err != nil {
			return err
		}
		tmpPath := filepath.Join(cfg.TmpDir, it.remote)
		if err := downloadFile(url, tmpPath); err != nil {
			return fmt.Errorf("download %s failed: %w", it.remote, err)
		}
		dstPath := filepath.Join(cfg.BinDir, it.localFile)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, dstPath); err != nil {
			return err
		}
		cfgKV[it.cfgKey] = geoVer
		changed = true
	}

	if changed {
		return writeKV(cfgPath, cfgKV)
	}
	return nil
}

func runUpdateCore(cfg startctl.Config) error {
	cfgPath := filepath.Join(cfg.CrashDir, "configs", "ShellCrash.cfg")
	cfgKV, err := parseKV(cfgPath)
	if err != nil {
		return err
	}
	remoteVer, err := fetchRemoteVersionKV(cfg, cfgKV)
	if err != nil {
		return err
	}

	crashCore := strings.TrimSpace(stripQuotes(cfgKV["crashcore"]))
	if crashCore == "" {
		crashCore = cfg.CrashCore
	}
	if crashCore == "" {
		crashCore = "meta"
	}
	coreKey := crashCore + "_v"
	coreVNew := strings.TrimSpace(remoteVer[coreKey])
	coreVOld := strings.TrimSpace(stripQuotes(cfgKV["core_v"]))
	if coreVNew == "" || coreVNew == coreVOld {
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新内核】中止-未检测到版本更新")
		return nil
	}

	if err := os.MkdirAll(cfg.TmpDir, 0o755); err != nil {
		return err
	}

	zipType := strings.TrimSpace(stripQuotes(cfgKV["zip_type"]))
	custLink := strings.TrimSpace(stripQuotes(cfgKV["custcorelink"]))
	if custLink != "" {
		switch {
		case strings.HasSuffix(custLink, ".tar.gz"):
			zipType = "tar.gz"
		case strings.HasSuffix(custLink, ".gz"):
			zipType = "gz"
		case strings.HasSuffix(custLink, ".upx"):
			zipType = "upx"
		}
	}
	if zipType == "" {
		zipType = "tar.gz"
	}

	target := "clash"
	if strings.Contains(crashCore, "singbox") {
		target = "singbox"
	}

	coreURL := custLink
	if coreURL == "" {
		cpucore := strings.TrimSpace(stripQuotes(cfgKV["cpucore"]))
		if cpucore == "" {
			cpucore = detectCPUCore()
		}
		if cpucore == "" {
			return fmt.Errorf("unable to determine cpucore")
		}
		asset := filepath.ToSlash(filepath.Join("bin", crashCore, target+"-linux-"+cpucore+"."+zipType))
		coreURL, err = resolveProjectAssetURL(cfg.CrashDir, cfgKV, asset)
		if err != nil {
			return err
		}
	}

	coreTmpArchive := filepath.Join(cfg.TmpDir, "Coretmp."+zipType)
	if err := downloadFile(coreURL, coreTmpArchive); err != nil {
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新内核】出错-下载失败！")
		return err
	}

	newCore := filepath.Join(cfg.TmpDir, "core_new")
	_ = os.Remove(newCore)
	if err := extractCoreBinary(coreTmpArchive, zipType, newCore); err != nil {
		_ = os.Remove(coreTmpArchive)
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新内核】出错-内核校验失败！")
		_ = startActionRun(cfg.CrashDir, "start")
		return err
	}

	coreV, command, err := validateCoreBinary(newCore, crashCore)
	if err != nil {
		_ = os.Remove(coreTmpArchive)
		_ = os.Remove(newCore)
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新内核】出错-内核校验失败！")
		_ = startActionRun(cfg.CrashDir, "start")
		return err
	}

	_ = os.Remove(filepath.Join(cfg.BinDir, "CrashCore.tar.gz"))
	_ = os.Remove(filepath.Join(cfg.BinDir, "CrashCore.gz"))
	_ = os.Remove(filepath.Join(cfg.BinDir, "CrashCore.upx"))

	binArchive := filepath.Join(cfg.BinDir, "CrashCore."+zipType)
	if err := os.MkdirAll(cfg.BinDir, 0o755); err != nil {
		return err
	}
	if err := moveFile(coreTmpArchive, binArchive); err != nil {
		return err
	}

	finalCore := filepath.Join(cfg.TmpDir, "CrashCore")
	if zipType == "upx" {
		if err := copyFile(binArchive, finalCore, 0o755); err != nil {
			return err
		}
		_ = os.Remove(newCore)
	} else {
		if err := moveFile(newCore, finalCore); err != nil {
			return err
		}
		if err := os.Chmod(finalCore, 0o755); err != nil {
			return err
		}
	}

	cfgKV["crashcore"] = crashCore
	cfgKV["core_v"] = coreV
	cfgKV["custcorelink"] = custLink
	if err := writeKV(cfgPath, cfgKV); err != nil {
		return err
	}
	if err := updateCommandEnv(cfg.CrashDir, command); err != nil {
		return err
	}

	_ = writeTaskLog(cfg.TmpDir, "任务【自动更新内核】下载完成，正在重启服务！")
	return startActionRun(cfg.CrashDir, "start")
}

func runUpdateScripts(cfg startctl.Config) error {
	cfgPath := filepath.Join(cfg.CrashDir, "configs", "ShellCrash.cfg")
	cfgKV, err := parseKV(cfgPath)
	if err != nil {
		return err
	}

	versionURL, err := resolveProjectAssetURL(cfg.CrashDir, cfgKV, "version")
	if err != nil {
		return err
	}
	versionBody, err := downloadURL(versionURL)
	if err != nil {
		return err
	}
	remoteVersion := parseScriptVersion(versionBody)
	localVersion := readLocalVersion(cfg.CrashDir)
	if remoteVersion == "" || (localVersion != "" && remoteVersion == localVersion) {
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新脚本】中止-未检测到版本更新")
		return nil
	}

	archiveURL, err := resolveProjectAssetURL(cfg.CrashDir, cfgKV, "ShellCrash.tar.gz")
	if err != nil {
		return err
	}
	archivePath := filepath.Join(cfg.TmpDir, "ShellCrash.tar.gz")
	if err := downloadFile(archiveURL, archivePath); err != nil {
		_ = os.Remove(archivePath)
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新脚本】出错-下载失败！")
		return err
	}
	defer os.Remove(archivePath)

	if err := startActionRun(cfg.CrashDir, "stop"); err != nil {
		return err
	}
	if err := extractTarGzIntoDir(archivePath, cfg.CrashDir); err != nil {
		_ = writeTaskLog(cfg.TmpDir, "任务【自动更新脚本】出错-解压失败！")
		_ = startActionRun(cfg.CrashDir, "start")
		return err
	}
	if err := initctlRun(initctl.Options{CrashDir: cfg.CrashDir, TmpDir: cfg.TmpDir}); err != nil {
		return err
	}
	return startActionRun(cfg.CrashDir, "start")
}

func detectCPUCore() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "amd64"
	case "386":
		return "386"
	case "arm64":
		return "arm64"
	case "arm":
		return "armv7"
	case "mipsle":
		return "mipsle-softfloat"
	case "mips":
		return "mips-softfloat"
	default:
		return ""
	}
}

func extractCoreBinary(src, zipType, dst string) error {
	switch zipType {
	case "tar.gz":
		return extractFromTarGz(src, dst)
	case "gz":
		return extractFromGzip(src, dst)
	case "upx":
		return copyFile(src, dst, 0o755)
	default:
		return moveFile(src, dst)
	}
}

func extractFromGzip(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, gr); err != nil {
		return err
	}
	return nil
}

func extractFromTarGz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	namePattern := regexp.MustCompile(`(?i)(CrashCore|sing|meta|mihomo|clash|pre)`)
	var selected []byte
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h == nil || h.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(h.Name)
		if !namePattern.MatchString(base) {
			continue
		}
		if h.Size <= 2000 {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		selected = b
	}
	if len(selected) == 0 {
		return fmt.Errorf("core executable not found in archive")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, selected, 0o755); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}

func extractTarGzIntoDir(src, dstDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	root := filepath.Clean(dstDir)
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h == nil {
			continue
		}
		name := strings.TrimPrefix(filepath.Clean(h.Name), string(filepath.Separator))
		target := filepath.Join(root, name)
		if target != root && !strings.HasPrefix(target, root+string(filepath.Separator)) {
			return fmt.Errorf("archive entry escapes target dir: %s", h.Name)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(h.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseScriptVersion(data []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			key := strings.TrimSpace(strings.ToLower(k))
			if key == "versionsh" || key == "versionsh_l" || key == "version" {
				return stripQuotes(v)
			}
			continue
		}
		return line
	}
	return ""
}

func readLocalVersion(crashDir string) string {
	b, err := os.ReadFile(filepath.Join(crashDir, "version"))
	if err != nil {
		return ""
	}
	for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line != "" {
			return line
		}
	}
	return ""
}

func validateCoreBinary(path, crashCore string) (version string, command string, err error) {
	h, err := exec.Command(path, "-h").CombinedOutput()
	if err != nil {
		return "", "", err
	}
	out := string(h)
	if strings.Contains(crashCore, "singbox") {
		if !strings.Contains(out, "sing-box") {
			return "", "", fmt.Errorf("unexpected singbox help output")
		}
		vb, err := exec.Command(path, "version").CombinedOutput()
		if err != nil {
			return "", "", err
		}
		version = parseSingboxVersion(string(vb))
		if version == "" {
			return "", "", fmt.Errorf("unable to parse singbox version")
		}
		return version, `"$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons"`, nil
	}
	if !strings.Contains(out, "-t") {
		return "", "", fmt.Errorf("unexpected clash help output")
	}
	vb, err := exec.Command(path, "-v").CombinedOutput()
	if err != nil {
		return "", "", err
	}
	version = parseClashVersion(string(vb))
	if version == "" {
		return "", "", fmt.Errorf("unable to parse clash version")
	}
	return version, `"$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"`, nil
}

func parseClashVersion(s string) string {
	line := strings.TrimSpace(strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")[0])
	if line == "" {
		return ""
	}
	if idx := strings.Index(line, " linux"); idx > 0 {
		line = line[:idx]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func parseSingboxVersion(s string) string {
	for _, raw := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "sing-box version") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[2]
			}
		}
		if strings.HasPrefix(strings.ToLower(line), "version") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1]
			}
		}
	}
	return ""
}

func updateCommandEnv(crashDir, command string) error {
	if strings.TrimSpace(command) == "" {
		return nil
	}
	envPath := filepath.Join(crashDir, "configs", "command.env")
	envKV, err := parseKV(envPath)
	if err != nil {
		return err
	}
	envKV["COMMAND"] = command
	return writeKV(envPath, envKV)
}

func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyFile(src, dst, 0o644); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func fetchRemoteVersionKV(cfg startctl.Config, cfgKV map[string]string) (map[string]string, error) {
	url, err := resolveProjectAssetURL(cfg.CrashDir, cfgKV, "bin/version")
	if err != nil {
		return nil, err
	}
	b, err := downloadURL(url)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m, nil
}

func resolveProjectAssetURL(crashDir string, cfgKV map[string]string, relPath string) (string, error) {
	relPath = strings.TrimLeft(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return "", fmt.Errorf("empty asset path")
	}
	updateURL := strings.TrimSpace(stripQuotes(cfgKV["update_url"]))
	if updateURL == "" {
		updateURL = "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master"
	}
	urlID := strings.TrimSpace(stripQuotes(cfgKV["url_id"]))
	if urlID == "" {
		return strings.TrimRight(updateURL, "/") + "/" + relPath, nil
	}

	branch := strings.TrimSpace(stripQuotes(cfgKV["release_type"]))
	if branch == "" {
		branch = "master"
	}
	if strings.HasPrefix(relPath, "bin/") {
		branch = "update"
	} else if strings.HasPrefix(relPath, "public/") || strings.HasPrefix(relPath, "rules/") {
		branch = "dev"
	}

	base := lookupServerBase(crashDir, urlID)
	if base == "" {
		return "", fmt.Errorf("url_id %q not found in servers.list", urlID)
	}
	base = strings.TrimRight(base, "/")
	if urlID == "101" || urlID == "104" {
		return base + "@" + branch + "/" + relPath, nil
	}
	return base + "/" + branch + "/" + relPath, nil
}

func lookupServerBase(crashDir, id string) string {
	paths := []string{
		filepath.Join(crashDir, "configs", "servers.list"),
		filepath.Join(crashDir, "public", "servers.list"),
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == id {
				return fields[2]
			}
		}
	}
	return ""
}

func downloadFile(url, dst string) error {
	b, err := downloadURL(url)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func downloadURL(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := taskHTTPDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(resp.Body)
}

func parseKV(path string) (map[string]string, error) {
	m := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return nil, err
	}
	for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m, nil
}

func writeKV(path string, kv map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for _, k := range keys {
		v := strings.TrimSpace(kv[k])
		if strings.Contains(v, " ") && !strings.HasPrefix(v, "'") && !strings.HasPrefix(v, "\"") {
			v = "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(v)
		buf.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func resolveTaskCommand(crashDir, id string) (command string, name string, err error) {
	paths := []string{
		filepath.Join(crashDir, "task", "task.list"),
		filepath.Join(crashDir, "task", "task.user"),
		filepath.Join(crashDir, "public", "task.list"),
	}
	for _, p := range paths {
		f, openErr := os.Open(p)
		if openErr != nil {
			if errors.Is(openErr, os.ErrNotExist) {
				continue
			}
			return "", "", openErr
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "#", 3)
			if len(parts) != 3 || parts[0] != id {
				continue
			}
			_ = f.Close()
			return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), nil
		}
		if scanErr := s.Err(); scanErr != nil {
			_ = f.Close()
			return "", "", scanErr
		}
		_ = f.Close()
	}
	return "", "", fmt.Errorf("task id %s not found", id)
}

func writeTaskLog(tmpDir, line string) error {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(tmpDir, "ShellCrash.log")
	prefix := time.Now().Format("2006-01-02_15:04:05") + "~"
	entry := prefix + line

	var lines []string
	if b, err := os.ReadFile(path); err == nil {
		for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			raw = strings.TrimSpace(raw)
			if raw != "" {
				lines = append(lines, raw)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	lines = append(lines, entry)
	if len(lines) > 199 {
		if len(lines) > 20 {
			lines = lines[20:]
		}
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
