package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusSetCrashDirScriptDispatchesToGoBinaryAndExportsVars(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "set_crashdir.sh")

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
	fakeCtl := filepath.Join(fakeBin, "shellcrash-installpathctl")
	body := "#!/bin/sh\n" +
		"printf '%s' \"$*\" > \"$SC_MARKER\"\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--env-file\" ]; then\n" +
		"    shift\n" +
		"    printf \"dir='/tmp/Install'\\nCRASHDIR='/tmp/Install/ShellCrash'\\n\" > \"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake installpathctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; set_crashdir; echo \"$CRASHDIR|$dir\"")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"systype=mi_snapshot",
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run set_crashdir wrapper: %v, out=%s", err, string(out))
	}
	gotOut := strings.TrimSpace(string(out))
	if gotOut != "/tmp/Install/ShellCrash|/tmp/Install" {
		t.Fatalf("unexpected exported variables: got=%q", gotOut)
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	if !strings.Contains(got, "--systype mi_snapshot --env-file") || !strings.HasSuffix(got, "select") {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}
