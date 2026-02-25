package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenuScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menu.sh")

	td := t.TempDir()
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-menuctl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake menuctl binary: %v", err)
	}

	cmd := exec.Command("sh", script, "-s", "start")
	cmd.Env = append(os.Environ(),
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run menu wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	want := "--crashdir " + repoRoot + " -s start"
	if got := strings.TrimSpace(string(gotBytes)); got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
}
