package startctl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"shellcrash/internal/coreconfig"
	"shellcrash/internal/firewall"
	"shellcrash/internal/lifecycle"
)

var controllerHTTPDo = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	return client.Do(req)
}
var controllerCoreConfigRun = coreconfig.Run

var beforeStartPingHosts = []string{"223.5.5.5", "1.2.4.8", "dns.alidns.com", "doh.pub"}
var beforeStartSleep = time.Sleep
var beforeStartPing = func(host string) bool {
	return exec.Command("ping", "-c", "3", host).Run() == nil
}
var beforeStartHasCommand = hasCommand
var beforeStartModprobeTun = func() {
	_ = exec.Command("modprobe", "tun").Run()
}

type Controller struct {
	Cfg     Config
	Runtime Runtime
}

func (c Controller) Run(action string, debugLevel string, debugFlash bool) error {
	return c.RunWithArgs(action, debugLevel, debugFlash, nil)
}

func (c Controller) RunWithArgs(action string, debugLevel string, debugFlash bool, extraArgs []string) error {
	switch action {
	case "start":
		return c.Start()
	case "stop":
		return c.Stop()
	case "bfstart":
		return c.beforeStart()
	case "afstart":
		return c.afterStart(false)
	case "start_firewall":
		return c.startFirewall()
	case "stop_firewall":
		return c.stopFirewall()
	case "restart":
		if err := c.Stop(); err != nil {
			return err
		}
		return c.Start()
	case "debug":
		return c.Debug(debugLevel, debugFlash)
	case "daemon":
		return c.Daemon()
	case "init":
		return c.generalInit()
	case "cronset":
		return c.cronset(extraArgs)
	case "core_exchange":
		return c.coreExchange(extraArgs)
	case "check_core":
		return c.checkCore()
	case "clash_check", "singbox_check", "clash_config_check", "singbox_config_check", "check_config":
		return c.corePrecheck()
	case "start_error":
		return c.handleStartError()
	case "check_network":
		return c.checkNetworkReachability()
	case "check_geo":
		return c.checkGeo(extraArgs)
	case "check_cnip":
		return c.checkCNIP(extraArgs)
	case "fw_start":
		return c.startFirewall()
	case "fw_stop":
		return c.stopFirewall()
	case "core_precheck":
		return c.corePrecheck()
	case "prepare_runtime":
		return c.corePrecheck()
	case "hotupdate":
		return c.hotUpdate()
	case "bot_tg_start":
		return c.botTGStart()
	case "bot_tg_stop":
		return c.botTGStop()
	case "bot_tg_cron":
		return c.botTGCron()
	default:
		if action == "" {
			return fmt.Errorf("missing action")
		}
		return c.runLegacyAction(action, extraArgs)
	}
}

func (c Controller) Start() error {
	if isRunning("CrashCore") {
		_ = c.Stop()
	}

	_ = c.stopFirewall()
	_ = os.Remove(filepath.Join(c.Cfg.CrashDir, ".start_error"))

	switch {
	case c.Cfg.FirewallArea == "5":
		return firewall.Start(c.Cfg.CrashDir)
	case c.Cfg.StartOld == "ON":
		return c.startLegacy()
	case c.Runtime.HasProcd:
		return runCommand("/etc/init.d/shellcrash", "start")
	case c.Runtime.IsSystemd():
		if c.Runtime.HasSystemdShellCrashUnit() {
			_ = runCommand("systemctl", "daemon-reload")
		}
		if err := runCommand("systemctl", "start", "shellcrash.service"); err != nil {
			_ = c.handleStartError()
			return err
		}
		return nil
	case c.Runtime.HasS6:
		if err := c.beforeStart(); err != nil {
			return err
		}
		if err := c.ensureShellCrashUser(); err != nil {
			return err
		}
		c.ensureTunModule()
		if err := (&c).runCorePreChecks(c.coreConfigPath()); err != nil {
			return err
		}
		if err := runCommand("/command/s6-svc", "-u", "/run/service/shellcrash"); err != nil {
			return err
		}
		if !fileExists(filepath.Join(c.Cfg.CrashDir, ".dis_startup")) {
			_ = os.WriteFile("/etc/s6-overlay/s6-rc.d/user/contents.d/afstart", []byte{}, 0o644)
		}
		go func() { _ = c.afterStart(false) }()
		return nil
	case c.Runtime.HasOpenRC:
		_ = runCommand("rc-service", "shellcrash", "stop")
		return runCommand("rc-service", "shellcrash", "start")
	default:
		return c.startLegacy()
	}
}

func (c Controller) Stop() error {
	if c.Cfg.ShellCrashPID != "" {
		if pid, err := strconv.Atoi(c.Cfg.ShellCrashPID); err == nil {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
		_ = os.Remove(filepath.Join(c.Cfg.TmpDir, "shellcrash.pid"))
		_ = c.stopFirewall()
	} else {
		switch {
		case c.Runtime.IsSystemd():
			_ = runCommand("systemctl", "stop", "shellcrash.service")
		case c.Runtime.HasProcd:
			_ = runCommand("/etc/init.d/shellcrash", "stop")
		case c.Runtime.HasS6:
			_ = runCommand("/command/s6-svc", "-d", "/run/service/shellcrash")
			_ = c.stopFirewall()
		case c.Runtime.HasOpenRC:
			_ = runCommand("rc-service", "shellcrash", "stop")
		default:
			_ = c.stopFirewall()
		}
	}

	_ = exec.Command("killall", "CrashCore").Run()
	_ = os.RemoveAll(filepath.Join(c.Cfg.TmpDir, "CrashCore"))
	return nil
}

func (c Controller) Debug(level string, flash bool) error {
	if isRunning("CrashCore") {
		_ = c.Stop()
	}
	_ = c.stopFirewall()
	if err := c.ensureCoreConfig(); err != nil {
		return err
	}
	if err := c.beforeStart(); err != nil {
		return err
	}
	if err := c.ensureShellCrashUser(); err != nil {
		return err
	}
	c.ensureTunModule()
	if err := (&c).runCorePreChecks(c.coreConfigPath()); err != nil {
		return err
	}

	if level != "" {
		target := filepath.Join(c.Cfg.TmpDir, "debug.log")
		if flash {
			target = filepath.Join(c.Cfg.CrashDir, "debug.log")
		}
		if err := c.startDetachedCore(c.expandCommand(), target, false); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	} else {
		if err := c.startDetachedCore(c.expandCommand(), "", false); err != nil {
			return err
		}
	}

	return c.afterStart(false)
}

func (c Controller) Daemon() error {
	marker := filepath.Join(c.Cfg.TmpDir, "crash_start_time")
	if fileExists(marker) {
		return c.Start()
	}
	time.Sleep(60 * time.Second)
	return os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
}

func (c Controller) startLegacy() error {
	if err := c.ensureCoreConfig(); err != nil {
		return err
	}
	if err := c.beforeStart(); err != nil {
		return err
	}
	if err := c.ensureShellCrashUser(); err != nil {
		return err
	}
	c.ensureTunModule()
	if err := (&c).runCorePreChecks(c.coreConfigPath()); err != nil {
		return err
	}
	cmdText := c.expandCommand()
	if cmdText == "" {
		return fmt.Errorf("COMMAND is empty")
	}

	var cmd *exec.Cmd
	if hasCommand("su") && passwdHasShellCrash() {
		cmd = exec.Command("su", "shellcrash", "-c", "nohup "+cmdText+" >/dev/null 2>&1 & echo $! > "+shellEscape(filepath.Join(c.Cfg.TmpDir, "shellcrash.pid")))
		cmd.Env = c.commandEnv()
		if err := cmd.Run(); err != nil {
			return err
		}
	} else {
		if err := c.startDetachedCore(cmdText, "", true); err != nil {
			return err
		}
	}

	go func() { _ = c.afterStart(false) }()
	return nil
}

func (c Controller) ensureCoreConfig() error {
	coreConfig := c.coreConfigPath()
	if fileExists(coreConfig) {
		return nil
	}
	_, err := controllerCoreConfigRun(coreconfig.Options{
		CrashDir: c.Cfg.CrashDir,
		TmpDir:   c.Cfg.TmpDir,
	})
	return err
}

func (c Controller) coreConfigPath() string {
	format := "yaml"
	if strings.Contains(c.Cfg.CrashCore, "singbox") {
		format = "json"
	}
	return filepath.Join(c.Cfg.CrashDir, format+"s", "config."+format)
}

func (c Controller) beforeStart() error {
	if err := c.checkNetworkReachability(); err != nil {
		return err
	}
	return lifecycle.BeforeStart(lifecycle.BeforeStartOptions{
		CrashDir:   c.Cfg.CrashDir,
		BinDir:     c.Cfg.BinDir,
		TmpDir:     c.Cfg.TmpDir,
		CoreConfig: c.coreConfigPath(),
		Host:       c.Cfg.Host,
		MixPort:    c.Cfg.MixPort,
		URL:        c.Cfg.URL,
		HTTPS:      c.Cfg.HTTPS,
	}, lifecycle.BeforeStartDeps{
		EnsureCoreConfig: c.ensureCoreConfig,
		RunTaskScript: func(path string) error {
			return c.runScript(path)
		},
	})
}

func (c Controller) checkNetworkReachability() error {
	if strings.EqualFold(c.Cfg.NetworkCheck, "OFF") {
		return nil
	}
	if fileExists(filepath.Join(c.Cfg.TmpDir, "crash_start_time")) {
		return nil
	}
	if !beforeStartHasCommand("ping") {
		return nil
	}
	for i, target := range beforeStartPingHosts {
		if beforeStartPing(target) {
			return nil
		}
		if i < len(beforeStartPingHosts)-1 {
			beforeStartSleep(5 * time.Second)
		}
	}
	return fmt.Errorf("network unreachable, startup stopped")
}

func (c Controller) ensureShellCrashUser() error {
	if !c.Runtime.IsRoot {
		return nil
	}
	if c.Cfg.FirewallArea != "2" && c.Cfg.FirewallArea != "3" && c.Runtime.InitName != "systemd" {
		return nil
	}
	if passwdHasShellCrash() {
		return nil
	}
	return rewriteUserFilesForShellCrash("/etc/passwd", "/etc/group")
}

func rewriteUserFilesForShellCrash(passwdPath, groupPath string) error {
	passwdLines := []string{}
	if b, err := os.ReadFile(passwdPath); err == nil {
		passwdLines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	groupLines := []string{}
	if b, err := os.ReadFile(groupPath); err == nil {
		groupLines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	passwdOut := filterLines(passwdLines, func(line string) bool {
		return strings.Contains(line, "0:7890") || strings.HasPrefix(line, "shellcrash:")
	})
	groupOut := filterLines(groupLines, func(line string) bool {
		return strings.Contains(line, "x:7890") || strings.HasPrefix(line, "shellcrash:")
	})

	passwdOut = append(passwdOut, "shellcrash:x:0:7890:::")
	groupOut = append(groupOut, "shellcrash:x:7890:")

	if err := os.WriteFile(passwdPath, []byte(strings.Join(passwdOut, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(groupPath, []byte(strings.Join(groupOut, "\n")+"\n"), 0o644)
}

func filterLines(lines []string, drop func(string) bool) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || drop(line) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func (c Controller) ensureTunModule() {
	if !strings.EqualFold(c.Cfg.RedirMod, "Tun") && !strings.EqualFold(c.Cfg.RedirMod, "Mix") {
		return
	}
	if !beforeStartHasCommand("modprobe") {
		return
	}
	beforeStartModprobeTun()
}

func (c Controller) stopFirewall() error {
	return firewall.Stop(c.Cfg.CrashDir)
}

func (c Controller) afterStart(ignoreCoreCheck bool) error {
	err := lifecycle.AfterStart(lifecycle.AfterStartOptions{
		CrashDir:      c.Cfg.CrashDir,
		BinDir:        c.Cfg.BinDir,
		TmpDir:        c.Cfg.TmpDir,
		StartOld:      c.Cfg.StartOld,
		StartDelaySec: c.Cfg.StartDelaySec,
		BotTGService:  c.Cfg.BotTGService,
	}, lifecycle.AfterStartDeps{
		IsCoreRunning: func() bool {
			if ignoreCoreCheck {
				return true
			}
			return c.isCoreReady()
		},
		StartFirewall: c.startFirewall,
		ReadSystemCron: func() ([]string, error) {
			return readSystemCron()
		},
		WriteSystemCron: func(lines []string) error {
			return writeSystemCron(lines)
		},
		RunTaskScript: func(path string) error {
			return c.runScript(path)
		},
	})
	if err == nil {
		go lifecycle.RestoreWebSelections(c.Cfg.CrashDir, c.Cfg.DBPort, c.Cfg.Secret)
		return nil
	}
	if errors.Is(err, lifecycle.ErrCoreNotRunning) {
		return c.handleStartError()
	}
	return err
}

func (c Controller) isCoreReady() bool {
	if c.checkControllerAPI() {
		return true
	}
	return isRunning("CrashCore")
}

func (c Controller) checkControllerAPI() bool {
	if c.Cfg.DBPort == "" {
		return false
	}
	u := "http://127.0.0.1:" + c.Cfg.DBPort + "/proxies"
	for i := 0; i < 30; i++ {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err == nil {
			if c.Cfg.Secret != "" {
				req.Header.Set("Authorization", "Bearer "+c.Cfg.Secret)
			}
			resp, err := controllerHTTPDo(req)
			if err == nil {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
				_ = resp.Body.Close()
				if resp.StatusCode < 400 && strings.Contains(string(body), "proxies") {
					return true
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func (c Controller) generalInit() error {
	return lifecycle.GeneralInit(c.Cfg.CrashDir, lifecycle.GeneralInitDeps{
		Start: c.Start,
		ReadSystemCron: func() ([]string, error) {
			return readSystemCron()
		},
		WriteSystemCron: func(lines []string) error {
			return writeSystemCron(lines)
		},
	})
}

var cronsetHasPersistentStore = func() bool {
	return fileExists("/jffs") || fileExists("/etc/storage/ShellCrash")
}

func (c Controller) cronset(args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("missing cronset keyword")
	}
	keyword := strings.TrimSpace(args[0])
	entry := strings.TrimSpace(strings.Join(args[1:], " "))

	lines, err := readSystemCron()
	if err != nil {
		return err
	}
	filtered := removeCronLinesContaining(lines, keyword)
	if entry != "" {
		filtered = append(filtered, entry)
	}
	if err := writeSystemCron(filtered); err != nil {
		return err
	}

	cronPath := filepath.Join(c.Cfg.CrashDir, "task", "cron")
	if cronsetHasPersistentStore() {
		if err := os.MkdirAll(filepath.Dir(cronPath), 0o755); err != nil {
			return err
		}
		content := strings.Join(filtered, "\n")
		if content != "" {
			content += "\n"
		}
		return os.WriteFile(cronPath, []byte(content), 0o644)
	}
	_ = os.Remove(cronPath)
	return nil
}

func removeCronLinesContaining(lines []string, keyword string) []string {
	if strings.TrimSpace(keyword) == "" {
		return append([]string{}, lines...)
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(line, keyword) {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func (c Controller) startFirewall() error {
	return firewall.Start(c.Cfg.CrashDir)
}

func (c Controller) coreExchange(args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("missing core_exchange target")
	}
	return c.switchCrashCore(strings.TrimSpace(args[0]))
}

func (c Controller) checkCore() error {
	return c.ensureCrashCoreBinary()
}

func (c Controller) corePrecheck() error {
	if err := c.ensureCoreConfig(); err != nil {
		return err
	}
	return (&c).runCorePreChecks(c.coreConfigPath())
}

func (c Controller) hotUpdate() error {
	res, err := controllerCoreConfigRun(coreconfig.Options{
		CrashDir: c.Cfg.CrashDir,
		TmpDir:   c.Cfg.TmpDir,
	})
	if err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(c.Cfg.TmpDir, "CrashCore"))

	format := res.Format
	if format == "" {
		format = "yaml"
	}
	configPath := res.CoreConfig
	if strings.TrimSpace(configPath) == "" {
		configPath = filepath.Join(c.Cfg.CrashDir, format+"s", "config."+format)
	}
	return c.hotReloadConfig(configPath)
}

func (c Controller) hotReloadConfig(configPath string) error {
	if strings.TrimSpace(c.Cfg.DBPort) == "" || strings.TrimSpace(configPath) == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"path": configPath})
	req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:"+c.Cfg.DBPort+"/configs", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.Cfg.Secret) != "" {
		req.Header.Set("Authorization", "Bearer "+c.Cfg.Secret)
	}
	resp, err := controllerHTTPDo(req)
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

func (c Controller) checkGeo(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("missing check_geo args")
	}
	return c.ensureGeoFile(strings.TrimSpace(args[0]), strings.TrimSpace(args[1]))
}

func (c Controller) checkCNIP(args []string) error {
	mode := ""
	if len(args) > 0 {
		mode = strings.TrimSpace(args[0])
	}
	return c.ensureCNIPAssetsForMode(mode, false)
}

func (c Controller) botTGStart() error {
	_ = exec.Command("killall", "bot_tg.sh").Run()
	_ = exec.Command("killall", "shellcrash-tgbot").Run()
	command, args, err := resolveTGBotExec(c.Cfg.CrashDir)
	if err != nil {
		return err
	}
	return c.startDetachedCommand(command, args, filepath.Join(c.Cfg.TmpDir, "bot_tg.pid"))
}

func (c Controller) botTGStop() error {
	_ = c.cronset([]string{"TG_BOT"})
	pidPath := filepath.Join(c.Cfg.TmpDir, "bot_tg.pid")
	if b, err := os.ReadFile(pidPath); err == nil {
		if pid, convErr := strconv.Atoi(strings.TrimSpace(string(b))); convErr == nil && pid > 0 {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
	_ = exec.Command("killall", "bot_tg.sh").Run()
	_ = exec.Command("killall", "shellcrash-tgbot").Run()
	_ = os.Remove(pidPath)
	return nil
}

func (c Controller) botTGCron() error {
	entry := fmt.Sprintf("* * * * * %s #ShellCrash-TG_BOT守护进程", c.watchdogCronCommand("bot_tg"))
	return c.cronset([]string{"TG_BOT守护进程", entry})
}

func (c Controller) runScript(path string) error {
	if !fileExists(path) {
		return nil
	}
	cmd := exec.Command("sh", path)
	cmd.Env = c.commandEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c Controller) commandEnv() []string {
	env := os.Environ()
	env = append(env,
		"CRASHDIR="+c.Cfg.CrashDir,
		"TMPDIR="+c.Cfg.TmpDir,
		"BINDIR="+c.Cfg.BinDir,
	)
	return env
}

func (c Controller) expandCommand() string {
	cmd := c.Cfg.Command
	if cmd == "" {
		return ""
	}
	cmd = strings.ReplaceAll(cmd, "$TMPDIR", c.Cfg.TmpDir)
	cmd = strings.ReplaceAll(cmd, "$BINDIR", c.Cfg.BinDir)
	return cmd
}

func isRunning(name string) bool {
	return exec.Command("pidof", name).Run() == nil
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func passwdHasShellCrash() bool {
	b, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte("shellcrash:x:0:7890"))
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveTGBotExec(crashDir string) (string, []string, error) {
	if p, err := exec.LookPath("shellcrash-tgbot"); err == nil {
		return p, []string{"--crashdir", crashDir}, nil
	}
	local := filepath.Join(crashDir, "bin", "shellcrash-tgbot")
	if fileExists(local) {
		return local, []string{"--crashdir", crashDir}, nil
	}
	if _, err := exec.LookPath("go"); err == nil {
		return "go", []string{"run", filepath.Join(crashDir, "cmd", "tgbot"), "--crashdir", crashDir}, nil
	}
	return "", nil, fmt.Errorf("shellcrash-tgbot is unavailable and go toolchain is not installed")
}

func readSystemCron() ([]string, error) {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		// Busybox returns non-zero when crontab is empty.
		if len(out) == 0 {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

func writeSystemCron(lines []string) error {
	f, err := os.CreateTemp("", "shellcrash-crontab-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return runCommand("crontab", f.Name())
}

func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (c Controller) watchdogCronCommand(target string) string {
	execPath := "shellcrash-startwatchdog"
	if strings.TrimSpace(c.Cfg.BinDir) != "" {
		execPath = filepath.Join(c.Cfg.BinDir, "shellcrash-startwatchdog")
	}
	return shellQuote(execPath) + " --crashdir " + shellQuote(c.Cfg.CrashDir) + " " + shellQuote(target)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (c Controller) startDetachedCore(cmdText string, logPath string, writePID bool) error {
	bin, args, err := splitCommandLine(cmdText)
	if err != nil {
		return err
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	stdout := io.Writer(devNull)
	stderr := io.Writer(devNull)
	var logFile *os.File
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		logFile = f
		defer logFile.Close()
		stdout = logFile
		stderr = logFile
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = c.commandEnv()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if hasCommand("setsid") {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if writePID {
		pidPath := filepath.Join(c.Cfg.TmpDir, "shellcrash.pid")
		if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644); err != nil {
			_ = cmd.Process.Kill()
			return err
		}
	}
	return cmd.Process.Release()
}

func (c Controller) startDetachedCommand(command string, args []string, pidFile string) error {
	if hasCommand("su") && passwdHasShellCrash() {
		quoted := make([]string, 0, len(args)+1)
		quoted = append(quoted, shellEscape(command))
		for _, arg := range args {
			quoted = append(quoted, shellEscape(arg))
		}
		inner := "nohup " + strings.Join(quoted, " ") + " >/dev/null 2>&1 & echo $! > " + shellEscape(pidFile)
		cmd := exec.Command("su", "shellcrash", "-c", inner)
		cmd.Env = c.commandEnv()
		return cmd.Run()
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	cmd := exec.Command(command, args...)
	cmd.Env = c.commandEnv()
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	if hasCommand("setsid") {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	return cmd.Process.Release()
}

func splitCommandLine(s string) (string, []string, error) {
	text := strings.TrimSpace(s)
	if text == "" {
		return "", nil, fmt.Errorf("COMMAND is empty")
	}
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
		case unicode.IsSpace(r) && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if escaped || inSingle || inDouble {
		return "", nil, fmt.Errorf("invalid COMMAND quoting")
	}
	flush()
	if len(args) == 0 {
		return "", nil, fmt.Errorf("COMMAND is empty")
	}
	return args[0], args[1:], nil
}

func (c Controller) runLegacyAction(action string, extraArgs []string) error {
	trimmed := strings.TrimSpace(action)
	if trimmed == "" {
		return fmt.Errorf("missing action")
	}
	if strings.Contains(trimmed, "..") || strings.ContainsAny(trimmed, `/\`) || strings.ContainsAny(trimmed, " \t\r\n;&|`$><(){}") {
		return fmt.Errorf("unsupported action %q", action)
	}
	legacyScript := filepath.Join(c.Cfg.CrashDir, "starts", trimmed+".sh")
	if !fileExists(legacyScript) {
		return fmt.Errorf("unsupported action %q", action)
	}
	args := append([]string{legacyScript}, extraArgs...)
	cmd := exec.Command("sh", args...)
	cmd.Env = c.commandEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
