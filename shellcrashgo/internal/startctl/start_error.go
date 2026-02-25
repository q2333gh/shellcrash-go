package startctl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var startErrorSleep = time.Sleep

func (c Controller) handleStartError() error {
	logPath := filepath.Join(c.Cfg.TmpDir, "core_test.log")
	_ = os.MkdirAll(c.Cfg.TmpDir, 0o755)

	if !strings.EqualFold(c.Cfg.StartOld, "ON") && hasCommand("journalctl") {
		out, _ := exec.Command("journalctl", "-u", "shellcrash").CombinedOutput()
		_ = os.WriteFile(logPath, out, 0o644)
	} else {
		_ = exec.Command("killall", "-9", "CrashCore").Run()
		_ = c.runCoreCommandForErrorLog(logPath)
	}

	_ = os.WriteFile(filepath.Join(c.Cfg.CrashDir, ".start_error"), []byte{}, 0o644)
	if logData, err := os.ReadFile(logPath); err == nil {
		if msg := extractCoreErrorSnippet(string(logData)); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
	}
	fmt.Fprintf(os.Stderr, "服务启动失败！请查看报错信息！详细信息请查看%s/core_test.log\n", c.Cfg.TmpDir)
	_ = c.Stop()
	return nil
}

func (c Controller) runCoreCommandForErrorLog(logPath string) error {
	cmdText := c.expandCommand()
	if cmdText == "" {
		return nil
	}
	bin, args, err := splitCommandLine(cmdText)
	if err != nil {
		return err
	}
	f, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command(bin, args...)
	cmd.Env = c.commandEnv()
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Start(); err != nil {
		return err
	}
	startErrorSleep(2 * time.Second)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_, _ = cmd.Process.Wait()
	return nil
}

func extractCoreErrorSnippet(logText string) string {
	re := regexp.MustCompile(`(?im).*?(error|fatal).*`)
	matches := re.FindAllString(logText, 5)
	for i := range matches {
		matches[i] = strings.TrimSpace(matches[i])
	}
	return strings.Join(matches, "\n")
}
