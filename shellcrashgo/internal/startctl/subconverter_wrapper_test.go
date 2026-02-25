package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusSubconverterScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "subconverter.sh")

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
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake coreconfig binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; subconverter; gen_link_flt; gen_link_ele; gen_link_config; gen_link_server; set_sub_ua")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run subconverter wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	want := strings.Join([]string{
		"--crashdir " + crashDir + " subconverter",
		"--crashdir " + crashDir + " subconverter-exclude",
		"--crashdir " + crashDir + " subconverter-include",
		"--crashdir " + crashDir + " subconverter-rule",
		"--crashdir " + crashDir + " subconverter-server",
		"--crashdir " + crashDir + " subconverter-ua",
	}, "\n")
	if got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
}
