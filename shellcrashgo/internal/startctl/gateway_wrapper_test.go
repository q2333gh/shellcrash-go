package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusGatewayScriptFWWanDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "7_gateway.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "menus"), 0o755); err != nil {
		t.Fatalf("mkdir menus: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "menus", "check_port.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write check_port stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "gen_base64.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write gen_base64 stub: %v", err)
	}
	gtCfg := filepath.Join(td, "gateway.cfg")
	if err := os.WriteFile(gtCfg, []byte(""), 0o644); err != nil {
		t.Fatalf("write gateway cfg: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-gatewayctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake gatewayctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; set_fw_wan")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"GT_CFG_PATH="+gtCfg,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run gateway wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" fw-wan" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusGatewayScriptVmessDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "7_gateway.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "menus"), 0o755); err != nil {
		t.Fatalf("mkdir menus: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "menus", "check_port.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write check_port stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "gen_base64.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write gen_base64 stub: %v", err)
	}
	gtCfg := filepath.Join(td, "gateway.cfg")
	if err := os.WriteFile(gtCfg, []byte(""), 0o644); err != nil {
		t.Fatalf("write gateway cfg: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-gatewayctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake gatewayctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; set_vmess")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"GT_CFG_PATH="+gtCfg,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run gateway wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" vmess" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusGatewayScriptSSSDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "7_gateway.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "menus"), 0o755); err != nil {
		t.Fatalf("mkdir menus: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "menus", "check_port.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write check_port stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "gen_base64.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write gen_base64 stub: %v", err)
	}
	gtCfg := filepath.Join(td, "gateway.cfg")
	if err := os.WriteFile(gtCfg, []byte(""), 0o644); err != nil {
		t.Fatalf("write gateway cfg: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-gatewayctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake gatewayctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; set_shadowsocks")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"GT_CFG_PATH="+gtCfg,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run gateway wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" sss" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusGatewayScriptTailscaleDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "7_gateway.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "menus"), 0o755); err != nil {
		t.Fatalf("mkdir menus: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "menus", "check_port.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write check_port stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "gen_base64.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write gen_base64 stub: %v", err)
	}
	gtCfg := filepath.Join(td, "gateway.cfg")
	if err := os.WriteFile(gtCfg, []byte(""), 0o644); err != nil {
		t.Fatalf("write gateway cfg: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-gatewayctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake gatewayctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; set_tailscale")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"GT_CFG_PATH="+gtCfg,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run gateway wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" tailscale" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}

func TestMenusGatewayScriptWireGuardDispatchesToGoBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "7_gateway.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "menus"), 0o755); err != nil {
		t.Fatalf("mkdir menus: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "libs"), 0o755); err != nil {
		t.Fatalf("mkdir libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "menus", "check_port.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write check_port stub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "libs", "gen_base64.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write gen_base64 stub: %v", err)
	}
	gtCfg := filepath.Join(td, "gateway.cfg")
	if err := os.WriteFile(gtCfg, []byte(""), 0o644); err != nil {
		t.Fatalf("write gateway cfg: %v", err)
	}
	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeCtl := filepath.Join(fakeBin, "shellcrash-gatewayctl")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeCtl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake gatewayctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; set_wireguard")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"GT_CFG_PATH="+gtCfg,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run gateway wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "--crashdir "+crashDir+" wireguard" {
		t.Fatalf("unexpected dispatch args: got=%q", got)
	}
}
