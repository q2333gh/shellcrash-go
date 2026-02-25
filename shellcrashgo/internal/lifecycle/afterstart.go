package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var ErrCoreNotRunning = errors.New("crashcore not running")

type AfterStartOptions struct {
	CrashDir      string
	BinDir        string
	TmpDir        string
	StartOld      string
	StartDelaySec int
	BotTGService  string
}

type AfterStartDeps struct {
	Sleep            func(time.Duration)
	Now              func() time.Time
	IsCoreRunning    func() bool
	StartFirewall    func() error
	ReadSystemCron   func() ([]string, error)
	WriteSystemCron  func([]string) error
	RunTaskScript    func(path string) error
	InjectFirewallFn func(firewallInitPath string, taskPath string) error
}

func AfterStart(opts AfterStartOptions, deps AfterStartDeps) error {
	if opts.TmpDir == "" || opts.CrashDir == "" {
		return fmt.Errorf("CrashDir/TmpDir required")
	}
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.IsCoreRunning == nil {
		return fmt.Errorf("IsCoreRunning dependency is required")
	}
	if deps.StartFirewall == nil {
		return fmt.Errorf("StartFirewall dependency is required")
	}
	if deps.ReadSystemCron == nil {
		deps.ReadSystemCron = func() ([]string, error) { return nil, nil }
	}
	if deps.WriteSystemCron == nil {
		deps.WriteSystemCron = func([]string) error { return nil }
	}
	if deps.RunTaskScript == nil {
		deps.RunTaskScript = func(string) error { return nil }
	}
	if deps.InjectFirewallFn == nil {
		deps.InjectFirewallFn = InjectFirewallTaskHook
	}

	marker := filepath.Join(opts.TmpDir, "crash_start_time")
	if _, err := os.Stat(marker); err != nil && opts.StartDelaySec > 0 {
		deps.Sleep(time.Duration(opts.StartDelaySec) * time.Second)
	}

	if !deps.IsCoreRunning() {
		return ErrCoreNotRunning
	}

	if err := deps.StartFirewall(); err != nil {
		return err
	}

	if err := os.MkdirAll(opts.TmpDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("%d\n", deps.Now().Unix())), 0o644); err != nil {
		return err
	}

	cron, _ := deps.ReadSystemCron()
	lines := append([]string{}, cron...)
	for _, p := range []string{filepath.Join(opts.CrashDir, "task", "cron"), filepath.Join(opts.CrashDir, "task", "running")} {
		if b, err := os.ReadFile(p); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				lines = append(lines, strings.TrimSpace(line))
			}
		}
	}
	lines = removeCronKeyword(lines, "start_legacy_wd.sh")
	if strings.EqualFold(opts.BotTGService, "ON") {
		lines = append(lines, fmt.Sprintf("* * * * * %s #ShellCrash-TG_BOT守护进程", watchdogCronCommand(opts.CrashDir, opts.BinDir, "bot_tg")))
	}
	if strings.EqualFold(opts.StartOld, "ON") {
		lines = append(lines, fmt.Sprintf("* * * * * %s #ShellCrash保守模式守护进程", watchdogCronCommand(opts.CrashDir, opts.BinDir, "shellcrash")))
	}
	merged := dedupeNonEmpty(lines)
	if err := deps.WriteSystemCron(merged); err != nil {
		return err
	}

	afstartTask := filepath.Join(opts.CrashDir, "task", "afstart")
	if stat, err := os.Stat(afstartTask); err == nil && stat.Size() > 0 {
		_ = deps.RunTaskScript(afstartTask)
	}

	affwTask := filepath.Join(opts.CrashDir, "task", "affirewall")
	fwInit := "/etc/init.d/firewall"
	if taskInfo, err := os.Stat(affwTask); err == nil && taskInfo.Size() > 0 {
		if _, err := os.Stat(fwInit); err == nil {
			if _, err := os.Stat(fwInit + ".bak"); err != nil {
				_ = deps.InjectFirewallFn(fwInit, affwTask)
			}
		}
	}

	return nil
}

func watchdogCronCommand(crashDir string, binDir string, target string) string {
	execPath := "shellcrash-startwatchdog"
	if strings.TrimSpace(binDir) != "" {
		execPath = filepath.Join(binDir, "shellcrash-startwatchdog")
	}
	return shellQuote(execPath) + " --crashdir " + shellQuote(crashDir) + " " + shellQuote(target)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func dedupeNonEmpty(lines []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}

func InjectFirewallTaskHook(firewallInitPath string, taskPath string) error {
	b, err := os.ReadFile(firewallInitPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	insert := ". " + taskPath
	if strings.Contains(string(b), insert) {
		return nil
	}

	restartRe := regexp.MustCompile(`fw.* restart`)
	startRe := regexp.MustCompile(`fw.* start`)

	out := make([]string, 0, len(lines)+2)
	for _, line := range lines {
		out = append(out, line)
		if restartRe.MatchString(line) || startRe.MatchString(line) {
			out = append(out, insert)
		}
	}
	return os.WriteFile(firewallInitPath, []byte(strings.Join(out, "\n")), 0o755)
}

type GeneralInitDeps struct {
	Start           func() error
	Sleep           func(time.Duration)
	ReadSystemCron  func() ([]string, error)
	WriteSystemCron func([]string) error
}

func GeneralInit(crashDir string, deps GeneralInitDeps) error {
	if deps.Start == nil {
		return fmt.Errorf("Start dependency is required")
	}
	if crashDir == "" {
		return fmt.Errorf("CrashDir required")
	}
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	alias, shType, zipType := readGeneralInitConfig(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	_ = configureProfileEnv(crashDir, alias, shType, zipType, deps.Sleep)

	if fileExists(filepath.Join(crashDir, ".dis_startup")) || fileExists(filepath.Join(crashDir, ".start_error")) {
		if deps.ReadSystemCron != nil && deps.WriteSystemCron != nil {
			lines, err := deps.ReadSystemCron()
			if err == nil {
				filtered := removeCronKeyword(lines, "保守模式守护进程")
				_ = deps.WriteSystemCron(filtered)
			}
		}
		return nil
	}
	return deps.Start()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readGeneralInitConfig(path string) (alias string, shType string, zipType string) {
	alias = "crash"
	shType = "sh"
	data, err := os.ReadFile(path)
	if err != nil {
		return alias, shType, zipType
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := stripConfigValue(v)
		switch key {
		case "my_alias":
			if val != "" {
				alias = val
			}
		case "shtype":
			if val != "" {
				shType = val
			}
		case "zip_type":
			zipType = val
		}
	}
	return alias, shType, zipType
}

func stripConfigValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func configureProfileEnv(crashDir string, alias string, shType string, zipType string, sleepFn func(time.Duration)) error {
	profile := "/etc/profile"
	if fileExists("/etc/storage/clash") || fileExists("/etc/storage/ShellCrash") {
		for i := 1; i < 10 && !isWritableFile(profile); i++ {
			sleepFn(3 * time.Second)
		}
		if !isWritableFile(profile) {
			profile = "/etc_ro/profile"
		}
		if zipType != "upx" {
			_ = exec.Command("mount", "-t", "tmpfs", "-o", "remount,rw,size=45M", "tmpfs", "/tmp").Run()
		}
	} else if fileExists("/jffs") {
		sleepFn(60 * time.Second)
		if !isWritableFile(profile) {
			if p := detectJFFSProfilePath(profile); p != "" {
				profile = p
			}
		}
	}
	return updateProfile(profile, crashDir, alias, shType)
}

func isWritableFile(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func detectJFFSProfilePath(profile string) string {
	data, err := os.ReadFile(profile)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`-f\s+(\S*jffs\S*profile)`)
	m := re.FindStringSubmatch(string(data))
	if len(m) != 2 {
		return ""
	}
	return m[1]
}

func updateProfile(profilePath string, crashDir string, alias string, shType string) error {
	if alias == "" {
		alias = "crash"
	}
	if shType == "" {
		shType = "sh"
	}
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines)+2)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "ShellCrash/menu.sh") {
			continue
		}
		if strings.HasPrefix(trimmed, "export CRASHDIR=") {
			continue
		}
		out = append(out, line)
	}
	out = append(out, fmt.Sprintf("alias %s=\"%s %s/menu.sh\"", alias, shType, crashDir))
	out = append(out, fmt.Sprintf("export CRASHDIR=\"%s\"", crashDir))
	return os.WriteFile(profilePath, []byte(strings.Join(out, "\n")), 0o644)
}

func removeCronKeyword(lines []string, keyword string) []string {
	if keyword == "" {
		return append([]string{}, lines...)
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, keyword) {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
