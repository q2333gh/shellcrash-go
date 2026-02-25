package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusUpgradeScriptSetServerDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "9_upgrade.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	stub := "#!/bin/sh\n"
	for _, name := range []string{"check_dir_avail.sh", "check_cpucore.sh", "web_get_bin.sh"} {
		if err := os.WriteFile(filepath.Join(crashDir, "libs", name), []byte(stub), 0o644); err != nil {
			t.Fatalf("write lib stub %s: %v", name, err)
		}
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-upgradectl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake upgradectl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; setserver")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run upgrade wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" setserver" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusUpgradeScriptSetCRTDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "9_upgrade.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	stub := "#!/bin/sh\n"
	for _, name := range []string{"check_dir_avail.sh", "check_cpucore.sh", "web_get_bin.sh"} {
		if err := os.WriteFile(filepath.Join(crashDir, "libs", name), []byte(stub), 0o644); err != nil {
			t.Fatalf("write lib stub %s: %v", name, err)
		}
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-upgradectl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake upgradectl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; setcrt")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run upgrade wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" setcrt" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusUpgradeScriptSetGeoDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "9_upgrade.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	stub := "#!/bin/sh\n"
	for _, name := range []string{"check_dir_avail.sh", "check_cpucore.sh", "web_get_bin.sh"} {
		if err := os.WriteFile(filepath.Join(crashDir, "libs", name), []byte(stub), 0o644); err != nil {
			t.Fatalf("write lib stub %s: %v", name, err)
		}
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-upgradectl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake upgradectl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; setgeo")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run upgrade wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" setgeo" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusUpgradeScriptSetDBDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "9_upgrade.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	stub := "#!/bin/sh\n"
	for _, name := range []string{"check_dir_avail.sh", "check_cpucore.sh", "web_get_bin.sh"} {
		if err := os.WriteFile(filepath.Join(crashDir, "libs", name), []byte(stub), 0o644); err != nil {
			t.Fatalf("write lib stub %s: %v", name, err)
		}
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-upgradectl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake upgradectl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; setdb")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run upgrade wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" setdb" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusUpgradeScriptUpgradeDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "9_upgrade.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-upgradectl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake upgradectl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; upgrade")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run upgrade wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" upgrade" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusUpgradeScriptSetCoreDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "9_upgrade.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-upgradectl")
	if err := os.WriteFile(fakeCtl, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake upgradectl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; setcore")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run upgrade wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" setcore" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}
