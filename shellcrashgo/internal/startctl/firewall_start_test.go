package startctl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartFirewallPrefersGoForIPTables(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	// Make config unreadable as a file to force firewall.Start() error.
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "starts"), 0o755); err != nil {
		t.Fatalf("mkdir starts: %v", err)
	}
	script := filepath.Join(crashDir, "starts", "fw_start.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	c := Controller{Cfg: Config{CrashDir: crashDir, FirewallMod: "iptables"}}
	if err := c.startFirewall(); err == nil {
		t.Fatalf("expected Go firewall start error for iptables")
	}
}

func TestStartFirewallUsesGoForNFTables(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	// Make config unreadable as a file to force firewall.Start() error.
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	c := Controller{Cfg: Config{CrashDir: crashDir, FirewallMod: "nftables"}}
	if err := c.startFirewall(); err == nil {
		t.Fatalf("expected Go firewall start error for nftables")
	}
}

func TestStartGatewayDoesNotFallbackToShellScript(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	// Force firewall.Start config read failure.
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "starts"), 0o755); err != nil {
		t.Fatalf("mkdir starts: %v", err)
	}
	marker := filepath.Join(td, "fallback_marker")
	script := filepath.Join(crashDir, "starts", "fw_start.sh")
	body := "#!/bin/sh\n" +
		"echo used > " + marker + "\n" +
		"exit 0\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	c := Controller{
		Cfg:     Config{CrashDir: crashDir, FirewallArea: "5"},
		Runtime: Runtime{},
	}
	if err := c.Start(); err == nil {
		t.Fatalf("expected Go firewall start error")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("fw_start.sh fallback should not run, marker=%v", err)
	}
}
