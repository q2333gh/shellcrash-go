package watchdog

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"shellcrash/internal/startctl"
)

type Options struct {
	CrashDir string
	Target   string
}

type Deps struct {
	IsProcessAlive  func(pid int) bool
	StartShellCrash func(crashDir string) error
	StartBotTG      func(crashDir string, target string, pidFile string) error
}

func Run(opts Options, deps Deps) error {
	target := strings.TrimSpace(opts.Target)
	if target == "" {
		return fmt.Errorf("watchdog target required")
	}
	if target != "shellcrash" && target != "bot_tg" {
		return fmt.Errorf("unsupported watchdog target %q", target)
	}
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	if deps.IsProcessAlive == nil {
		deps.IsProcessAlive = defaultIsProcessAlive
	}
	if deps.StartShellCrash == nil {
		deps.StartShellCrash = defaultStartShellCrash
	}
	if deps.StartBotTG == nil {
		deps.StartBotTG = defaultStartBotTG
	}

	tmpDir := "/tmp/ShellCrash"
	pidFile := filepath.Join(tmpDir, target+".pid")
	lockDir := filepath.Join(tmpDir, "start_"+target+".lock")

	if fileExists(filepath.Join(crashDir, ".start_error")) && !fileExists(filepath.Join(tmpDir, "crash_start_time")) {
		return fmt.Errorf("startup blocked by .start_error marker")
	}

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	defer func() { _ = os.Remove(lockDir) }()

	if pidAlive, err := checkPIDAlive(pidFile, deps.IsProcessAlive); err != nil {
		return err
	} else if pidAlive {
		return nil
	}

	if target == "shellcrash" {
		return deps.StartShellCrash(crashDir)
	}
	return deps.StartBotTG(crashDir, target, pidFile)
}

func checkPIDAlive(pidFile string, isAlive func(int) bool) (bool, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		_ = os.Remove(pidFile)
		return false, nil
	}
	if isAlive(pid) {
		return true, nil
	}
	_ = os.Remove(pidFile)
	return false, nil
}

func defaultIsProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true
	}
	_, statErr := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return statErr == nil
}

func defaultStartShellCrash(crashDir string) error {
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	return ctl.Start()
}

func defaultStartBotTG(crashDir string, target string, pidFile string) error {
	_ = exec.Command("killall", "bot_tg.sh").Run()
	_ = exec.Command("killall", "shellcrash-tgbot").Run()
	if target == "bot_tg" {
		command, args, err := resolveTGBotExec(crashDir)
		if err != nil {
			return err
		}
		return startDetached(command, args, pidFile)
	}
	return fmt.Errorf("unsupported bot target %q", target)
}

func resolveTGBotExec(crashDir string) (string, []string, error) {
	if p, err := exec.LookPath("shellcrash-tgbot"); err == nil {
		return p, []string{"--crashdir", crashDir}, nil
	}
	local := filepath.Join(crashDir, "bin", "shellcrash-tgbot")
	if _, err := os.Stat(local); err == nil {
		return local, []string{"--crashdir", crashDir}, nil
	}
	if _, err := exec.LookPath("go"); err == nil {
		return "go", []string{"run", filepath.Join(crashDir, "cmd", "tgbot"), "--crashdir", crashDir}, nil
	}
	return "", nil, fmt.Errorf("shellcrash-tgbot is unavailable and go toolchain is not installed")
}

func startDetached(command string, args []string, pidFile string) error {
	if hasCommand("su") && passwdHasShellCrash() {
		quotedArgs := make([]string, 0, len(args)+1)
		quotedArgs = append(quotedArgs, shellEscape(command))
		for _, arg := range args {
			quotedArgs = append(quotedArgs, shellEscape(arg))
		}
		inner := "nohup " + strings.Join(quotedArgs, " ") + " >/dev/null 2>&1 & echo $! > " + shellEscape(pidFile)
		return exec.Command("su", "shellcrash", "-c", inner).Run()
	}
	return startDetachedDirect(command, args, pidFile)
}

func startDetachedDirect(command string, args []string, pidFile string) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	cmd := exec.Command(command, args...)
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

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func passwdHasShellCrash() bool {
	b, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "shellcrash:x:0:7890")
}

func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
