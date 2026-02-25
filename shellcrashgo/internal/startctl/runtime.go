package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runtime struct {
	User      string
	InitName  string
	HasOpenRC bool
	HasProcd  bool
	HasS6     bool
	IsRoot    bool
}

func DetectRuntime() Runtime {
	r := Runtime{User: os.Getenv("USER")}
	if r.User == "" {
		r.User = os.Getenv("LOGNAME")
	}
	r.IsRoot = os.Geteuid() == 0

	if b, err := os.ReadFile("/proc/1/comm"); err == nil {
		r.InitName = strings.TrimSpace(string(b))
	}
	r.HasProcd = fileExists("/etc/rc.common") && r.InitName == "procd"
	r.HasS6 = strings.Contains(r.InitName, "s6")
	r.HasOpenRC = exec.Command("rc-status", "-r").Run() == nil
	return r
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r Runtime) IsSystemd() bool {
	return r.User == "root" && r.InitName == "systemd"
}

func (r Runtime) HasSystemdShellCrashUnit() bool {
	out, err := exec.Command("systemctl", "show", "-p", "FragmentPath", "shellcrash").Output()
	if err != nil {
		return false
	}
	line := strings.TrimSpace(strings.TrimPrefix(string(out), "FragmentPath="))
	if line == "" {
		return false
	}
	_, err = os.Stat(filepath.Clean(line))
	return err == nil
}
