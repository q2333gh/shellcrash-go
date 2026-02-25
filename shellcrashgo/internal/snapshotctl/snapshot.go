package snapshotctl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Options struct {
	CrashDir string
	FSRoot   string
	Action   string
}

type Deps struct {
	RunCommand       func(name string, args ...string) error
	CommandOutput    func(name string, args ...string) ([]byte, error)
	HasCommand       func(name string) bool
	ModInfoIPTables  func() string
	IsProcessRunning func(name string) bool
	HasLANInterface  func(fsRoot string) bool
	Sleep            func(time.Duration)
}

func Run(opts Options, deps Deps) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	fsRoot := strings.TrimSpace(opts.FSRoot)
	if fsRoot == "" {
		fsRoot = "/"
	}
	action := strings.TrimSpace(opts.Action)
	if action == "" {
		action = "default"
	}
	if deps.RunCommand == nil {
		deps.RunCommand = func(name string, args ...string) error { return exec.Command(name, args...).Run() }
	}
	if deps.CommandOutput == nil {
		deps.CommandOutput = func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		}
	}
	if deps.HasCommand == nil {
		deps.HasCommand = func(name string) bool {
			_, err := exec.LookPath(name)
			return err == nil
		}
	}
	if deps.ModInfoIPTables == nil {
		deps.ModInfoIPTables = defaultModInfoIPTables
	}
	if deps.IsProcessRunning == nil {
		deps.IsProcessRunning = func(name string) bool { return exec.Command("pidof", name).Run() == nil }
	}
	if deps.HasLANInterface == nil {
		deps.HasLANInterface = defaultHasLANInterface
	}
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}

	ctx := runCtx{opts: Options{CrashDir: crashDir, FSRoot: fsRoot, Action: action}, deps: deps}
	switch action {
	case "tunfix":
		return ctx.tunFix()
	case "tproxyfix":
		return ctx.tproxyFix()
	case "auto_clean":
		return ctx.autoClean()
	case "init":
		return ctx.init()
	case "default":
		if deps.IsProcessRunning("CrashCore") {
			return nil
		}
		return ctx.init()
	default:
		return fmt.Errorf("unsupported snapshot action %q", action)
	}
}

type runCtx struct {
	opts Options
	deps Deps
}

func (r runCtx) hostPath(abs string) string {
	abs = "/" + strings.TrimPrefix(abs, "/")
	if r.opts.FSRoot == "/" {
		return abs
	}
	return filepath.Join(r.opts.FSRoot, strings.TrimPrefix(abs, "/"))
}

func (r runCtx) tunFix() error {
	modulePath := strings.TrimSpace(r.deps.ModInfoIPTables())
	if modulePath == "" {
		return fmt.Errorf("unable to locate ip_tables.ko")
	}
	moduleDir := filepath.Dir(modulePath)
	if err := os.MkdirAll(r.hostPath("/tmp/overlay/upper"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(r.hostPath("/tmp/overlay/work"), 0o755); err != nil {
		return err
	}

	lower := r.hostPath(moduleDir)
	if err := r.deps.RunCommand(
		"mount",
		"-o",
		"noatime,lowerdir="+lower+",upperdir="+r.hostPath("/tmp/overlay/upper")+",workdir="+r.hostPath("/tmp/overlay/work"),
		"-t",
		"overlay",
		"overlay_mods_only",
		lower,
	); err != nil {
		return err
	}
	src := filepath.Join(r.opts.CrashDir, "tools", "tun.ko")
	dst := filepath.Join(lower, "tun.ko")
	_ = os.Remove(dst)
	return os.Symlink(src, dst)
}

func (r runCtx) tproxyFix() error {
	path := r.hostPath("/etc/init.d/qca-nss-ecm")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	text := strings.ReplaceAll(string(b), "sysctl -w net.bridge.bridge-nf-call-ip", "#sysctl -w net.bridge.bridge-nf-call-ip")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return err
	}
	if err := r.deps.RunCommand("sysctl", "-w", "net.bridge.bridge-nf-call-iptables=0"); err != nil {
		return err
	}
	return r.deps.RunCommand("sysctl", "-w", "net.bridge.bridge-nf-call-ip6tables=0")
}

func (r runCtx) autoClean() error {
	_ = os.RemoveAll(r.hostPath("/data/etc_bak"))
	_ = os.RemoveAll(r.hostPath("/data/usr/log"))
	_ = os.RemoveAll(r.hostPath("/data/usr/sec_cfg"))
	_ = r.deps.RunCommand(r.hostPath("/etc/init.d/stat_points"), "stop")
	_ = r.deps.RunCommand(r.hostPath("/etc/init.d/stat_points"), "disable")
	return commentCronKeywords(r.hostPath("/etc/crontabs/root"), []string{"logrotate", "sec_cfg_bak"})
}

func (r runCtx) init() error {
	_ = r.deps.RunCommand(r.hostPath("/etc/init.d/shellcrash"), "disable")
	_ = r.pruneLegacyWatchdogCron()
	if err := r.waitForConfigReady(); err != nil {
		return err
	}

	for i := 0; i < 18; i++ {
		if r.deps.HasLANInterface(r.opts.FSRoot) {
			break
		}
		r.deps.Sleep(10 * time.Second)
	}
	if err := r.autoSSH(); err != nil {
		return err
	}
	if err := r.autoClean(); err != nil {
		return err
	}
	if err := r.autoStart(); err != nil {
		return err
	}
	r.startScriptIfExists("/data/auto_start.sh")
	r.startScriptIfExists("/data/auto_ssh/auto_ssh.sh")
	return nil
}

func (r runCtx) waitForConfigReady() error {
	cfgPath := filepath.Join(r.opts.CrashDir, "configs", "ShellCrash.cfg")
	i := 0
	for !fileExists(cfgPath) {
		if i > 20 {
			return fmt.Errorf("snapshot init timeout waiting for config: %s", cfgPath)
		}
		i++
		r.deps.Sleep(3 * time.Second)
	}
	return nil
}

func (r runCtx) pruneLegacyWatchdogCron() error {
	cronPath := r.hostPath("/etc/crontabs/root")
	b, err := os.ReadFile(cronPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	next := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "start_legacy_wd.sh shellcrash") {
			continue
		}
		next = append(next, line)
	}
	out := strings.Join(next, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(cronPath, []byte(out), 0o644)
}

func (r runCtx) autoSSH() error {
	cfg := parseSimpleKV(filepath.Join(r.opts.CrashDir, "configs", "ShellCrash.cfg"))

	channel := strings.TrimSpace(r.commandOutputOrEmpty("uci", "-c", "/usr/share/xiaoqiang", "get", "xiaoqiang_version.version.CHANNEL"))
	if channel != "stable" {
		_ = r.deps.RunCommand("uci", "-c", "/usr/share/xiaoqiang", "set", "xiaoqiang_version.version.CHANNEL=stable")
		_ = r.deps.RunCommand("uci", "-c", "/usr/share/xiaoqiang", "commit", "xiaoqiang_version.version")
	}

	dropbearRunning := r.deps.IsProcessRunning("dropbear")
	hasPort22 := strings.Contains(r.commandOutputOrEmpty("netstat", "-ntul"), ":22")
	if !dropbearRunning || !hasPort22 {
		if err := rewriteDropbearChannelDebug(r.hostPath("/etc/init.d/dropbear")); err != nil {
			return err
		}
		_ = r.deps.RunCommand(r.hostPath("/etc/init.d/dropbear"), "restart")
		pwd := strings.TrimSpace(cfg["mi_autoSSH_pwd"])
		if pwd == "" {
			pwd = strings.TrimSpace(cfg["mi_mi_autoSSH_pwd"])
		}
		if pwd != "" {
			cmd := fmt.Sprintf("printf '%%s\\n%%s\\n' %s %s | passwd root", shellQuote(pwd), shellQuote(pwd))
			_ = r.deps.RunCommand("sh", "-c", cmd)
		}
	}

	sshEn := strings.TrimSpace(r.commandOutputOrEmpty("nvram", "get", "ssh_en"))
	if sshEn == "0" {
		_ = r.deps.RunCommand("nvram", "set", "ssh_en=1")
	}
	telnetEn := strings.TrimSpace(r.commandOutputOrEmpty("nvram", "get", "telnet_en"))
	if telnetEn == "0" {
		_ = r.deps.RunCommand("nvram", "set", "telnet_en=1")
	}
	_ = r.deps.RunCommand("nvram", "commit")

	if err := os.MkdirAll(r.hostPath("/etc/dropbear"), 0o755); err != nil {
		return err
	}
	for _, name := range []string{"dropbear_rsa_host_key", "authorized_keys"} {
		src := firstExisting(
			filepath.Join(r.opts.CrashDir, "configs", name),
			filepath.Join(r.opts.CrashDir, "tools", name),
		)
		if src == "" {
			continue
		}
		dst := r.hostPath(filepath.Join("/etc/dropbear", name))
		_ = os.Remove(dst)
		if err := os.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func (r runCtx) autoStart() error {
	initScript := r.hostPath("/etc/init.d/shellcrash")
	if _, err := os.Stat(initScript); os.IsNotExist(err) {
		src := filepath.Join(r.opts.CrashDir, "starts", "shellcrash.procd")
		b, readErr := os.ReadFile(src)
		if readErr == nil {
			if err := os.MkdirAll(filepath.Dir(initScript), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(initScript, b, 0o755); err != nil {
				return err
			}
		}
	}
	if fileExists(filepath.Join(r.opts.CrashDir, ".dis_startup")) || fileExists(filepath.Join(r.opts.CrashDir, ".start_error")) {
		return nil
	}

	cfg := parseSimpleKV(filepath.Join(r.opts.CrashDir, "configs", "ShellCrash.cfg"))
	if fileExists(filepath.Join(r.opts.CrashDir, "tools", "tun.ko")) {
		_ = r.tunFix()
	}
	if fileExists(r.hostPath("/etc/init.d/qca-nss-ecm")) && strings.Contains(strings.ToLower(cfg["redir_mod"]), "tproxy") {
		_ = r.tproxyFix()
	}
	if fileExists(filepath.Join(r.opts.CrashDir, "tools", "ca-certificates.crt")) {
		dst := r.hostPath("/etc/ssl/certs/ca-certificates.crt")
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = copyFile(filepath.Join(r.opts.CrashDir, "tools", "ca-certificates.crt"), dst, 0o644)
	}
	_ = r.deps.RunCommand(filepath.Join(r.opts.CrashDir, "start.sh"), "stop")
	if err := r.deps.RunCommand(initScript, "start"); err != nil {
		return err
	}
	return r.deps.RunCommand(initScript, "enable")
}

func (r runCtx) startScriptIfExists(absPath string) {
	path := r.hostPath(absPath)
	if !fileExists(path) {
		return
	}
	cmd := exec.Command("sh", path)
	_ = cmd.Start()
}

func (r runCtx) commandOutputOrEmpty(name string, args ...string) string {
	b, err := r.deps.CommandOutput(name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func rewriteDropbearChannelDebug(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	re := regexp.MustCompile(`(?m)^channel=.*$`)
	next := re.ReplaceAll(b, []byte(`channel="debug"`))
	if bytes.Equal(next, b) {
		return nil
	}
	return os.WriteFile(path, next, 0o755)
}

func defaultModInfoIPTables() string {
	out, err := exec.Command("modinfo", "ip_tables").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, raw := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "filename:") {
			continue
		}
		p := strings.TrimSpace(strings.TrimPrefix(line, "filename:"))
		if strings.HasSuffix(p, "/ip_tables.ko") {
			return p
		}
	}
	return ""
}

func defaultHasLANInterface(fsRoot string) bool {
	path := "/proc/net/dev"
	if fsRoot != "/" {
		path = filepath.Join(fsRoot, "proc", "net", "dev")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	return strings.Contains(string(b), "lan")
}

func commentCronKeywords(path string, keys []string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	for i, raw := range lines {
		trim := strings.TrimSpace(raw)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		for _, key := range keys {
			if strings.Contains(raw, key) {
				lines[i] = "#ShellCrash自动注释 " + raw
				break
			}
		}
	}
	out := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(out), 0o644)
}

func parseSimpleKV(path string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
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
		out[strings.TrimSpace(k)] = strings.TrimSpace(strings.Trim(v, `"'`))
	}
	return out
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func copyFile(src, dst string, mode os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, mode)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
