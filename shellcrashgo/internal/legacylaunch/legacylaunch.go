package legacylaunch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	Command string
	Name    string
	TmpDir  string
}

func Run(opts Options) error {
	command := strings.TrimSpace(opts.Command)
	name := strings.TrimSpace(opts.Name)
	if command == "" {
		return fmt.Errorf("command is required")
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}
	tmpDir := strings.TrimSpace(opts.TmpDir)
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	pidFile := filepath.Join(tmpDir, name+".pid")

	if hasCommand("su") && passwdHasShellCrash() {
		inner := "nohup " + shellEscape("sh") + " -c " + shellEscape(command) + " >/dev/null 2>&1 & echo $! > " + shellEscape(pidFile)
		return exec.Command("su", "shellcrash", "-c", inner).Run()
	}
	return startDetached(command, pidFile)
}

func startDetached(command, pidFile string) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	var cmd *exec.Cmd
	if hasCommand("setsid") {
		cmd = exec.Command("setsid", "sh", "-c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	return cmd.Process.Release()
}

var lookPath = exec.LookPath

func hasCommand(name string) bool {
	_, err := lookPath(name)
	return err == nil
}

var readFile = os.ReadFile

func passwdHasShellCrash() bool {
	b, err := readFile("/etc/passwd")
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
