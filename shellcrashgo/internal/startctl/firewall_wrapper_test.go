package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFWIPTablesScriptDispatchesToStartFirewall(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "starts", "fw_iptables.sh")
	runFWWrapperScriptTest(t, script, "start_iptables")
}

func TestFWNFTablesScriptDispatchesToStartFirewall(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "starts", "fw_nftables.sh")
	runFWWrapperScriptTest(t, script, "start_nftables")
}

func runFWWrapperScriptTest(t *testing.T, scriptPath string, fnName string) {
	t.Helper()

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	startsDir := filepath.Join(crashDir, "starts")
	if err := os.MkdirAll(startsDir, 0o755); err != nil {
		t.Fatalf("mkdir starts: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	startScript := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(filepath.Join(crashDir, "start.sh"), []byte(startScript), 0o755); err != nil {
		t.Fatalf("write start.sh: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; "+fnName+" one two")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+scriptPath,
		"SC_MARKER="+marker,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run wrapper script: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	want := "start_firewall one two"
	if got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/startctl/<this file> -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
