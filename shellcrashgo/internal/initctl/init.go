package initctl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var initctlHasCommand = func(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

var initctlRunCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

type Options struct {
	CrashDir string
	TmpDir   string
	FSRoot   string
}

func Run(opts Options) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	tmpDir := strings.TrimSpace(opts.TmpDir)
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	fsRoot := strings.TrimSpace(opts.FSRoot)
	if fsRoot == "" {
		fsRoot = "/"
	}

	if err := ensureBaseDirs(crashDir); err != nil {
		return err
	}
	cfgPath, err := ensureConfigFile(crashDir)
	if err != nil {
		return err
	}

	if err := migrateLegacyFiles(crashDir, cfgPath); err != nil {
		return err
	}
	if err := normalizeConfig(cfgPath); err != nil {
		return err
	}
	if err := ensureCommandEnv(crashDir, tmpDir, cfgPath); err != nil {
		return err
	}
	if err := configureServiceInstall(crashDir, cfgPath, fsRoot); err != nil {
		return err
	}
	if err := configureFirmwareExtras(crashDir, cfgPath, fsRoot); err != nil {
		return err
	}
	if err := ensureFirewallMode(cfgPath); err != nil {
		return err
	}
	if err := cleanupLegacyHostArtifacts(fsRoot); err != nil {
		return err
	}
	return nil
}

func ensureBaseDirs(crashDir string) error {
	dirs := []string{
		filepath.Join(crashDir, "configs"),
		filepath.Join(crashDir, "yamls"),
		filepath.Join(crashDir, "jsons"),
		filepath.Join(crashDir, "tools"),
		filepath.Join(crashDir, "task"),
		filepath.Join(crashDir, "ruleset"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ensureConfigFile(crashDir string) (string, error) {
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	if _, err := os.Stat(cfgPath); err == nil {
		return cfgPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.WriteFile(cfgPath, []byte("#ShellCrash配置文件，不明勿动！\n"), 0o644); err != nil {
		return "", err
	}
	return cfgPath, nil
}

func migrateLegacyFiles(crashDir, cfgPath string) error {
	oldCfg := filepath.Join(crashDir, "configs", "ShellClash.cfg")
	if exists(oldCfg) {
		if err := moveFile(oldCfg, cfgPath); err != nil {
			return err
		}
	}

	for _, name := range []string{"config.yaml.bak", "user.yaml", "proxies.yaml", "proxy-groups.yaml", "rules.yaml", "others.yaml"} {
		if err := moveFile(filepath.Join(crashDir, name), filepath.Join(crashDir, "yamls", name)); err != nil {
			return err
		}
	}
	configYaml := filepath.Join(crashDir, "config.yaml")
	if !isSymlink(configYaml) {
		if err := moveFile(configYaml, filepath.Join(crashDir, "yamls", "config.yaml")); err != nil {
			return err
		}
	}

	for _, name := range []string{"fake_ip_filter", "mac", "web_save", "servers.list", "fake_ip_filter.list", "fallback_filter.list", "singbox_providers.list", "clash_providers.list"} {
		if err := moveFile(filepath.Join(crashDir, name), filepath.Join(crashDir, "configs", name)); err != nil {
			return err
		}
	}

	for _, name := range []string{"dropbear_rsa_host_key", "authorized_keys", "tun.ko", "ShellDDNS.sh"} {
		if err := moveFile(filepath.Join(crashDir, name), filepath.Join(crashDir, "tools", name)); err != nil {
			return err
		}
	}
	for _, name := range []string{"cron", "task.list"} {
		if err := moveFile(filepath.Join(crashDir, name), filepath.Join(crashDir, "task", name)); err != nil {
			return err
		}
	}
	if err := moveFile(filepath.Join(crashDir, "menus", "task_cmd.sh"), filepath.Join(crashDir, "task", "task.sh")); err != nil {
		return err
	}
	if err := moveFile(filepath.Join(crashDir, "menus", "task_cmd_legacy.sh"), filepath.Join(crashDir, "task", "task_legacy.sh")); err != nil {
		return err
	}

	if err := moveFile(filepath.Join(crashDir, "geosite.dat"), filepath.Join(crashDir, "GeoSite.dat")); err != nil {
		return err
	}
	if err := moveFile(filepath.Join(crashDir, "ruleset", "geosite-cn.srs"), filepath.Join(crashDir, "ruleset", "cn.srs")); err != nil {
		return err
	}
	if err := moveFile(filepath.Join(crashDir, "ruleset", "geosite-cn.mrs"), filepath.Join(crashDir, "ruleset", "cn.mrs")); err != nil {
		return err
	}
	if err := moveBySuffix(crashDir, ".srs", filepath.Join(crashDir, "ruleset")); err != nil {
		return err
	}
	if err := moveBySuffix(crashDir, ".mrs", filepath.Join(crashDir, "ruleset")); err != nil {
		return err
	}

	for _, rel := range []string{
		"webget.sh",
		"misnap_init.sh",
		"core.new",
		"configs/ShellCrash.cfg.bak",
	} {
		_ = os.Remove(filepath.Join(crashDir, rel))
	}
	_ = os.RemoveAll(filepath.Join(crashDir, "rules"))
	return nil
}

func normalizeConfig(cfgPath string) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	text := string(data)
	replacements := []struct{ old, new string }{
		{"clashcore", "crashcore"},
		{"clash_v", "core_v"},
		{"clash.meta", "meta"},
		{"ShellClash", "ShellCrash"},
		{"cpucore=armv8", "cpucore=arm64"},
		{"redir_mod=Redir模式", "redir_mod=Redir"},
		{"redir_mod=Tproxy模式", "redir_mod=Tproxy"},
		{"redir_mod=Tun模式", "redir_mod=Tun"},
		{"redir_mod=混合模式", "redir_mod=Mix"},
		{"redir_mod=纯净模式", "firewall_area=4"},
	}
	for _, r := range replacements {
		text = strings.ReplaceAll(text, r.old, r.new)
	}

	lines := strings.Split(text, "\n")
	for i := range lines {
		trim := strings.TrimSpace(lines[i])
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		k, v, ok := strings.Cut(lines[i], "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch v {
		case "已启用", "已开启":
			lines[i] = k + "=ON"
		case "未启用", "未开启":
			lines[i] = k + "=OFF"
		}
	}
	out := strings.Join(lines, "\n")
	return os.WriteFile(cfgPath, []byte(out), 0o644)
}

func ensureCommandEnv(crashDir, tmpDir, cfgPath string) error {
	envPath := filepath.Join(crashDir, "configs", "command.env")
	envKV := map[string]string{}
	if kv, err := parseKVFile(envPath); err == nil {
		envKV = kv
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	cfgKV, err := parseKVFile(cfgPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	crashCore := stripQuotes(cfgKV["crashcore"])

	envKV["TMPDIR"] = tmpDir
	envKV["BINDIR"] = crashDir
	envKV["COMMAND"] = defaultCommandForCore(crashCore)
	return writeKVFile(envPath, envKV)
}

func configureServiceInstall(crashDir, cfgPath, fsRoot string) error {
	startsDir := filepath.Join(crashDir, "starts")

	procComm := strings.TrimSpace(readFileString(filepath.Join(fsRoot, "proc", "1", "comm")))
	procdMode := exists(filepath.Join(fsRoot, "etc", "rc.common")) && procComm == "procd"
	systemdMode := procComm == "systemd" && (os.Geteuid() == 0 || fsRoot != "/")
	openRCMode := detectOpenRC(fsRoot)

	switch {
	case procdMode:
		if err := copyIfExists(filepath.Join(startsDir, "shellcrash.procd"), filepath.Join(fsRoot, "etc", "init.d", "shellcrash"), 0o755); err != nil {
			return err
		}
	case systemdMode:
		if err := ensureSystemdServiceUser(fsRoot); err != nil {
			return err
		}
		targetDir := detectSystemdDir(fsRoot)
		if targetDir == "" {
			if err := setConfigValue(cfgPath, "start_old", "ON"); err != nil {
				return err
			}
			break
		}
		dst := filepath.Join(targetDir, "shellcrash.service")
		if err := copyWithReplace(filepath.Join(startsDir, "shellcrash.service"), dst, "/etc/ShellCrash", crashDir, 0o644); err != nil {
			return err
		}
		_ = exec.Command("systemctl", "daemon-reload").Run()
	case openRCMode:
		if err := copyIfExists(filepath.Join(startsDir, "shellcrash.openrc"), filepath.Join(fsRoot, "etc", "init.d", "shellcrash"), 0o755); err != nil {
			return err
		}
	default:
		if err := setConfigValue(cfgPath, "start_old", "ON"); err != nil {
			return err
		}
	}

	_ = os.Remove(filepath.Join(startsDir, "shellcrash.procd"))
	_ = os.Remove(filepath.Join(startsDir, "shellcrash.openrc"))
	_ = os.Remove(filepath.Join(startsDir, "shellcrash.service"))
	return nil
}

func cleanupLegacyHostArtifacts(fsRoot string) error {
	if err := removeLinesMatching(filepath.Join(fsRoot, "etc", "passwd"), func(line string) bool {
		return strings.Contains(line, "shellclash:")
	}); err != nil {
		return err
	}
	if err := removeLinesMatching(filepath.Join(fsRoot, "etc", "group"), func(line string) bool {
		return strings.Contains(line, "shellclash:")
	}); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(fsRoot, "etc", "init.d", "clash"))
	return nil
}

func ensureSystemdServiceUser(fsRoot string) error {
	passwd := filepath.Join(fsRoot, "etc", "passwd")
	group := filepath.Join(fsRoot, "etc", "group")

	if err := removeLinesMatching(passwd, func(line string) bool {
		return strings.HasPrefix(line, "shellcrash:") || strings.Contains(line, ":0:7890:") || strings.Contains(line, ":7890:7890:")
	}); err != nil {
		return err
	}
	if err := removeLinesMatching(group, func(line string) bool {
		return strings.HasPrefix(line, "shellcrash:") || strings.Contains(line, ":x:7890:")
	}); err != nil {
		return err
	}

	if err := appendLineIfMissing(passwd, "shellcrash:x:0:7890::/home/shellcrash:/bin/sh"); err != nil {
		return err
	}
	if err := appendLineIfMissing(group, "shellcrash:x:7890:"); err != nil {
		return err
	}
	return nil
}

func removeLinesMatching(path string, drop func(string) bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if errors.Is(err, os.ErrPermission) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if drop(line) {
			continue
		}
		out = append(out, line)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil
		}
		return err
	}
	return nil
}

func appendLineIfMissing(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := string(data)
	if strings.Contains(text, line) {
		return nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += line + "\n"
	return os.WriteFile(path, []byte(text), 0o644)
}

func detectSystemdDir(fsRoot string) string {
	for _, rel := range []string{"etc/systemd/system", "usr/lib/systemd/system"} {
		p := filepath.Join(fsRoot, rel)
		if isWritableDir(p) {
			return p
		}
	}
	return ""
}

func detectOpenRC(fsRoot string) bool {
	rcStatus := filepath.Join(fsRoot, "sbin", "rc-status")
	if _, err := os.Stat(rcStatus); err == nil {
		return true
	}
	return false
}

func readFileString(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func isWritableDir(path string) bool {
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		return false
	}
	f, err := os.CreateTemp(path, ".shellcrash-initctl-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func copyIfExists(src, dst string, mode os.FileMode) error {
	if !exists(src) {
		return nil
	}
	return copyWithReplace(src, dst, "", "", mode)
}

func copyWithReplace(src, dst, old, new string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	content := string(data)
	if old != "" {
		content = strings.ReplaceAll(content, old, new)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(f, content); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func defaultCommandForCore(core string) string {
	if strings.Contains(strings.ToLower(core), "singbox") {
		return "$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons"
	}
	return "$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"
}

func ensureFirewallMode(cfgPath string) error {
	kv, err := parseKVFile(cfgPath)
	if err != nil {
		return err
	}
	if _, ok := kv["firewall_mod"]; ok {
		return nil
	}
	mode := "iptables"
	if initctlHasCommand("nft") {
		mode = "nftables"
	}
	return setConfigValue(cfgPath, "firewall_mod", mode)
}

func configureFirmwareExtras(crashDir, cfgPath, fsRoot string) error {
	systype := detectSystemType(fsRoot)
	if systype != "" {
		if err := setConfigValue(cfgPath, "systype", systype); err != nil {
			return err
		}
	}
	switch systype {
	case "Padavan":
		if err := configureInitHook(crashDir, cfgPath, fsRoot, "/etc/storage/started_script.sh"); err != nil {
			return err
		}
	case "asusrouter":
		if initPath := detectAsusInitPath(fsRoot); initPath != "" {
			if err := configureInitHook(crashDir, cfgPath, fsRoot, initPath); err != nil {
				return err
			}
		}
	case "mi_snapshot", "ng_snapshot":
		if err := configureSnapshotInit(crashDir, fsRoot, systype); err != nil {
			return err
		}
	case "container":
		if err := configureContainerDefaults(crashDir, cfgPath, fsRoot); err != nil {
			return err
		}
	default:
		_ = os.Remove(filepath.Join(crashDir, "starts", "snapshot_init.sh"))
	}
	return nil
}

func configureSnapshotInit(crashDir, fsRoot, systype string) error {
	origScript := filepath.Join(crashDir, "starts", "snapshot_init.sh")
	if !exists(origScript) {
		return nil
	}
	_ = os.Chmod(origScript, 0o755)

	scriptPath := origScript
	if systype == "mi_snapshot" {
		scriptPath = filepath.Join(fsRoot, "data", "shellcrash_init.sh")
		if err := copyWithReplace(origScript, scriptPath, "", "", 0o755); err != nil {
			return err
		}
		if err := pinCrashDirInScript(scriptPath, crashDir); err != nil {
			return err
		}
		autoStart := filepath.Join(fsRoot, "data", "auto_start.sh")
		if !exists(autoStart) {
			if err := os.WriteFile(autoStart, []byte("#用于自定义需要开机启动的功能或者命令，会在开机后自动运行\n"), 0o644); err != nil {
				return err
			}
		}
	}

	runtimeScriptPath := pathFromFSRoot(scriptPath, fsRoot)
	if err := applySnapshotUCI(runtimeScriptPath); err != nil {
		return err
	}
	return nil
}

func pinCrashDirInScript(path, crashDir string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	assigned := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "CRASHDIR=") {
			lines[i] = "CRASHDIR=" + crashDir
			assigned = true
			break
		}
	}
	if !assigned {
		insertAt := 0
		if len(lines) > 0 && strings.HasPrefix(lines[0], "#!") {
			insertAt = 1
		}
		prefix := []string{"CRASHDIR=" + crashDir, "export CRASHDIR"}
		withPrefix := make([]string, 0, len(lines)+len(prefix))
		withPrefix = append(withPrefix, lines[:insertAt]...)
		withPrefix = append(withPrefix, prefix...)
		withPrefix = append(withPrefix, lines[insertAt:]...)
		lines = withPrefix
	}
	out := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(out), 0o755)
}

func pathFromFSRoot(path, fsRoot string) string {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(fsRoot)
	if cleanRoot == "/" {
		return cleanPath
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return cleanPath
	}
	return "/" + filepath.ToSlash(rel)
}

func applySnapshotUCI(scriptPath string) error {
	if !initctlHasCommand("uci") {
		return nil
	}
	ops := [][]string{
		{"delete", "firewall.auto_ssh"},
		{"delete", "firewall.ShellCrash"},
		{"set", "firewall.ShellCrash=include"},
		{"set", "firewall.ShellCrash.type=script"},
		{"set", "firewall.ShellCrash.path=" + scriptPath},
		{"set", "firewall.ShellCrash.enabled=1"},
		{"commit", "firewall"},
	}
	for _, op := range ops {
		if err := initctlRunCommand("uci", op...); err != nil {
			return err
		}
	}
	return nil
}

func detectSystemType(fsRoot string) string {
	if isContainerRuntime(fsRoot) {
		return "container"
	}
	if exists(filepath.Join(fsRoot, "etc", "storage", "started_script.sh")) {
		return "Padavan"
	}
	if exists(filepath.Join(fsRoot, "jffs")) {
		return "asusrouter"
	}
	if exists(filepath.Join(fsRoot, "data", "etc", "crontabs", "root")) {
		return "mi_snapshot"
	}
	if isWritableDir(filepath.Join(fsRoot, "var", "mnt", "cfg", "firewall")) {
		return "ng_snapshot"
	}
	return ""
}

func isContainerRuntime(fsRoot string) bool {
	if b, err := os.ReadFile(filepath.Join(fsRoot, "proc", "1", "cgroup")); err == nil {
		if strings.Contains(string(b), "/docker/") ||
			strings.Contains(string(b), "/lxc/") ||
			strings.Contains(string(b), "/kubepods/") ||
			strings.Contains(string(b), "/crio/") ||
			strings.Contains(string(b), "/containerd/") {
			return true
		}
	}
	if exists(filepath.Join(fsRoot, "run", ".containerenv")) {
		return true
	}
	if exists(filepath.Join(fsRoot, ".dockerenv")) {
		return true
	}
	return false
}

func detectAsusInitPath(fsRoot string) string {
	if exists(filepath.Join(fsRoot, "jffs", "scripts")) {
		return "/jffs/scripts/nat-start"
	}
	if exists(filepath.Join(fsRoot, "jffs", ".asusrouter")) {
		return "/jffs/.asusrouter"
	}
	return ""
}

func configureInitHook(crashDir, cfgPath, fsRoot, initPath string) error {
	target := filepath.Join(fsRoot, strings.TrimPrefix(initPath, "/"))
	line := crashDir + "/starts/general_init.sh & #ShellCrash初始化脚本"
	if err := appendManagedLine(target, line, "ShellCrash初始化"); err != nil {
		return err
	}
	_ = os.Chmod(target, 0o755)
	_ = os.Chmod(filepath.Join(crashDir, "starts", "general_init.sh"), 0o755)
	return setConfigValue(cfgPath, "initdir", initPath)
}

func appendManagedLine(path, line, marker string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	out := make([]string, 0, len(lines)+1)
	for _, raw := range lines {
		raw = strings.TrimSpace(raw)
		if raw == "" || (marker != "" && strings.Contains(raw, marker)) {
			continue
		}
		out = append(out, raw)
	}
	out = append(out, line)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func configureContainerDefaults(crashDir, cfgPath, fsRoot string) error {
	defaults := [][2]string{
		{"userguide", "1"},
		{"crashcore", "meta"},
		{"dns_mod", "mix"},
		{"firewall_area", "1"},
		{"firewall_mod", "nftables"},
		{"release_type", "master"},
		{"start_old", "OFF"},
	}
	for _, kv := range defaults {
		if err := setConfigValue(cfgPath, kv[0], kv[1]); err != nil {
			return err
		}
	}
	if err := appendLineIfMissing(filepath.Join(fsRoot, "etc", "profile"), crashDir+"/menu.sh"); err != nil {
		return err
	}
	wrapper := strings.Join([]string{
		"#!/bin/sh",
		"CRASHDIR=${CRASHDIR:-" + crashDir + "}",
		"export CRASHDIR",
		"exec \"$CRASHDIR/menu.sh\" \"$@\"",
		"",
	}, "\n")
	path := filepath.Join(fsRoot, "usr", "bin", "crash")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(wrapper), 0o755); err != nil {
		return err
	}
	return nil
}

func moveBySuffix(srcDir, suffix, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), suffix) {
			continue
		}
		if err := moveFile(filepath.Join(srcDir, ent.Name()), filepath.Join(dstDir, ent.Name())); err != nil {
			return err
		}
	}
	return nil
}

func moveFile(src, dst string) error {
	if !exists(src) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	_ = os.Remove(dst)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

func parseKVFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	kv := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return kv, nil
}

func writeKVFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func setConfigValue(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, raw := range lines {
		trim := strings.TrimSpace(raw)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		k, _, ok := strings.Cut(raw, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			found = true
		}
	}
	if !found {
		lines = append(lines, key+"="+value)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
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

func PrintSummary(opts Options) string {
	crashDir := opts.CrashDir
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	return fmt.Sprintf("ShellCrash init bootstrap completed at %s", crashDir)
}
