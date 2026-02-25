package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckCoreScriptDispatchesToCheckCore(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "starts", "check_core.sh")
	runStartWrapperFunctionTest(t, script, "check_core", []string{"foo", "bar"}, "check_core foo bar")
}

func TestCheckCoreScriptDirectExecDispatchesToCheckCore(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "starts", "check_core.sh")
	runStartWrapperDirectExecTest(t, script, []string{"foo", "bar"}, "check_core foo bar")
}

func TestCoreExchangeScriptDispatchesToCoreExchange(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "starts", "core_exchange.sh")
	runStartWrapperFunctionTest(t, script, "core_exchange", []string{"meta"}, "core_exchange meta")
}

func TestCoreExchangeScriptDirectExecDispatchesToCoreExchange(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "starts", "core_exchange.sh")
	runStartWrapperDirectExecTest(t, script, []string{"meta"}, "core_exchange meta")
}

func runStartWrapperFunctionTest(t *testing.T, scriptPath string, fnName string, args []string, want string) {
	t.Helper()
	crashDir, marker := prepareWrapperFixture(t)

	call := ". \"$SC_SCRIPT\"; " + fnName
	for _, arg := range args {
		call += " " + shellEscape(arg)
	}
	cmd := exec.Command("sh", "-c", call)
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+scriptPath,
		"SC_MARKER="+marker,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run sourced wrapper script: %v, out=%s", err, string(out))
	}
	assertWrapperDispatch(t, marker, want)
}

func runStartWrapperDirectExecTest(t *testing.T, scriptPath string, args []string, want string) {
	t.Helper()
	crashDir, marker := prepareWrapperFixture(t)

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	cmd.Env = append(os.Environ(), "CRASHDIR="+crashDir, "SC_MARKER="+marker)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run wrapper direct-exec: %v, out=%s", err, string(out))
	}
	assertWrapperDispatch(t, marker, want)
}

func prepareWrapperFixture(t *testing.T) (string, string) {
	t.Helper()
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatalf("mkdir crashdir: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	startScript := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(filepath.Join(crashDir, "start.sh"), []byte(startScript), 0o755); err != nil {
		t.Fatalf("write start.sh: %v", err)
	}
	return crashDir, marker
}

func assertWrapperDispatch(t *testing.T, marker string, want string) {
	t.Helper()
	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	if got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
}

