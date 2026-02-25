package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusStartScriptStartServiceDispatchesToStart(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "1_start.sh")
	runMenuStartWrapperFunctionTest(t, script, "start_service", "start")
}

func TestMenusStartScriptStartCoreDispatchesToStart(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "1_start.sh")
	runMenuStartWrapperFunctionTest(t, script, "start_core", "start")
}

func runMenuStartWrapperFunctionTest(t *testing.T, scriptPath, fnName, want string) {
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

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; "+fnName)
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+scriptPath,
		"SC_MARKER="+marker,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run menu start wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
}
