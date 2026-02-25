package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusToolsScriptSSHToolsDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; ssh_tools")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" ssh-tools" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusToolsScriptMiAutoSSHDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; mi_autoSSH")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" mi-auto-ssh" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusToolsScriptLogPusherDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; log_pusher")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" log-pusher" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusToolsScriptTestCommandDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; testcommand")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" testcommand" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusToolsScriptDDNSToolsDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-ddnsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake ddnsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; ddns_tools")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" menu" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusToolsScriptDebugDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; debug")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" debug" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusToolsScriptToolsDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "8_tools.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "logger.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write logger stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "web_get_bin.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write web_get_bin stub: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; tools")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run tools wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" tools" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusUserguideScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "userguide.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-toolsctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake toolsctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; userguide")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run userguide wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" userguide" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}
