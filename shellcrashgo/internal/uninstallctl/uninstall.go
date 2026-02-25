package uninstallctl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"shellcrash/internal/startctl"
)

type Options struct {
	CrashDir   string
	BinDir     string
	Alias      string
	KeepConfig bool
	FSRoot     string
}

type Deps struct {
	StartAction func(crashDir, action string, args []string) error
	RunCommand  func(name string, args ...string) error
}

func Run(opts Options, deps Deps) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = os.Getenv("CRASHDIR")
	}
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	if crashDir == "/" {
		return fmt.Errorf("invalid crashdir %q", crashDir)
	}
	binDir := strings.TrimSpace(opts.BinDir)
	if binDir == "" {
		binDir = os.Getenv("BINDIR")
	}
	if binDir == "" {
		binDir = crashDir
	}
	fsRoot := strings.TrimSpace(opts.FSRoot)
	if fsRoot == "" {
		fsRoot = "/"
	}

	if deps.StartAction == nil {
		deps.StartAction = runStartAction
	}
	if deps.RunCommand == nil {
		deps.RunCommand = func(name string, args ...string) error {
			return exec.Command(name, args...).Run()
		}
	}

	for _, item := range []struct {
		action string
		args   []string
	}{
		{action: "stop"},
		{action: "cronset", args: []string{"clash服务"}},
		{action: "cronset", args: []string{"订阅链接"}},
		{action: "cronset", args: []string{"ShellCrash初始化"}},
		{action: "cronset", args: []string{"task.sh"}},
	} {
		_ = deps.StartAction(crashDir, item.action, item.args)
	}

	if opts.KeepConfig {
		if err := keepConfigOnly(crashDir); err != nil {
			return err
		}
	} else {
		if err := os.RemoveAll(crashDir); err != nil {
			return err
		}
	}

	alias := strings.TrimSpace(opts.Alias)
	_ = filterLines(hostPath(fsRoot, "/etc/profile"), func(line string) bool {
		if alias != "" && strings.Contains(line, "alias "+alias+"=") {
			return true
		}
		if strings.Contains(line, "alias crash=") {
			return true
		}
		if strings.Contains(line, "export CRASHDIR=") || strings.Contains(line, "export crashdir=") {
			return true
		}
		if strings.Contains(line, "all_proxy") || strings.Contains(line, "ALL_PROXY") {
			return true
		}
		return false
	})
	_ = filterLines(hostPath(fsRoot, "/etc/firewall.user"), func(line string) bool {
		return strings.Contains(line, "启用外网访问SSH服务")
	})
	_ = filterLines(hostPath(fsRoot, "/etc/storage/started_script.sh"), func(line string) bool {
		return strings.Contains(line, "ShellCrash初始化")
	})
	_ = filterLines(hostPath(fsRoot, "/jffs/.asusrouter"), func(line string) bool {
		return strings.Contains(line, "ShellCrash初始化")
	})

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		_ = filterLines(filepath.Join(home, ".zshrc"), func(line string) bool {
			if alias != "" && strings.Contains(line, "alias "+alias+"=") {
				return true
			}
			return strings.Contains(line, "export CRASHDIR=")
		})
	}

	if binDir != "" && binDir != crashDir {
		_ = os.RemoveAll(binDir)
	}
	for _, p := range []string{
		"/etc/init.d/shellcrash",
		"/etc/systemd/system/shellcrash.service",
		"/usr/lib/systemd/system/shellcrash.service",
		"/www/clash",
		"/tmp/ShellCrash",
		"/usr/bin/crash",
	} {
		_ = os.RemoveAll(hostPath(fsRoot, p))
	}

	_ = filterLines(hostPath(fsRoot, "/etc/passwd"), func(line string) bool {
		return strings.Contains(line, "0:7890") || strings.HasPrefix(line, "shellcrash:")
	})
	_ = filterLines(hostPath(fsRoot, "/etc/group"), func(line string) bool {
		return strings.HasPrefix(line, "shellcrash:") || strings.Contains(line, ":7890:")
	})

	if hasCommand("userdel") {
		_ = deps.RunCommand("userdel", "-r", "shellcrash")
	}
	if hasCommand("nvram") {
		_ = deps.RunCommand("nvram", "set", "script_usbmount=")
		_ = deps.RunCommand("nvram", "commit")
	}
	return nil
}

func runStartAction(crashDir, action string, args []string) error {
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	return ctl.RunWithArgs(action, "", false, args)
}

func keepConfigOnly(crashDir string) error {
	tmpBase := filepath.Join("/tmp", "ShellCrash")
	if err := os.MkdirAll(tmpBase, 0o755); err != nil {
		return err
	}
	backup := map[string]string{}
	for _, name := range []string{"configs", "yamls", "jsons"} {
		src := filepath.Join(crashDir, name)
		if !exists(src) {
			continue
		}
		dst := filepath.Join(tmpBase, name+"_bak")
		_ = os.RemoveAll(dst)
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		backup[name] = dst
	}
	entries, err := os.ReadDir(crashDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(crashDir, entry.Name())); err != nil {
			return err
		}
	}
	for name, path := range backup {
		if err := os.Rename(path, filepath.Join(crashDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func filterLines(path string, remove func(line string) bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if remove(line) {
			continue
		}
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	text := strings.Join(out, "\n")
	if text != "" {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func hostPath(fsRoot, absPath string) string {
	absPath = "/" + strings.TrimPrefix(absPath, "/")
	if strings.TrimSpace(fsRoot) == "" || fsRoot == "/" {
		return absPath
	}
	return filepath.Join(fsRoot, strings.TrimPrefix(absPath, "/"))
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
