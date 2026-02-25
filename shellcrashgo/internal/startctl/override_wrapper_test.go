package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusOverrideScriptSetRulesDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "override.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatalf("mkdir crashdir: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-coreconfig")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake coreconfig binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; setrules")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run override wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" override-rules" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusOverrideScriptDispatchesExtendedFunctionsToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "override.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatalf("mkdir crashdir: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-coreconfig")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake coreconfig binary: %v", err)
	}

	tests := []struct {
		fn   string
		args string
	}{
		{fn: "override", args: "--crashdir " + crashDir + " override"},
		{fn: "setgroups", args: "--crashdir " + crashDir + " override-groups"},
		{fn: "setproxies", args: "--crashdir " + crashDir + " override-proxies"},
		{fn: "set_clash_adv", args: "--crashdir " + crashDir + " override-clash-adv"},
		{fn: "set_singbox_adv", args: "--crashdir " + crashDir + " override-singbox-adv"},
	}

	for _, tt := range tests {
		t.Run(tt.fn, func(t *testing.T) {
			if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
				t.Fatalf("remove marker: %v", err)
			}
			cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; "+tt.fn)
			cmd.Env = append(os.Environ(),
				"CRASHDIR="+crashDir,
				"SC_SCRIPT="+script,
				"SC_MARKER="+marker,
				"PATH="+fakeBin+":"+os.Getenv("PATH"),
			)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("run override wrapper function %s: %v, out=%s", tt.fn, err, string(out))
			}
			gotBytes, err := os.ReadFile(marker)
			if err != nil {
				t.Fatalf("read marker: %v", err)
			}
			if got := strings.TrimSpace(string(gotBytes)); got != tt.args {
				t.Fatalf("unexpected dispatch args: got=%q want=%q", got, tt.args)
			}
		})
	}
}
