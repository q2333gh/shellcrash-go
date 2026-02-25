package startctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaultsAndOverrides(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgData := "firewall_area=5\nfirewall_mod=nftables\ncn_ip_route=OFF\nipv6_redir=ON\ndb_port=10090\nsecret=s3\nstart_old=ON\nstart_delay=12\nbot_tg_service=ON\ncrashcore=singboxr\ndisoverride=1\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	envData := "BINDIR=/opt/shell\nTMPDIR=/tmp/custom\nCOMMAND='$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons'\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte(envData), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	if c.FirewallArea != "5" || c.StartOld != "ON" || c.CrashCore != "singboxr" {
		t.Fatalf("unexpected config fields: %+v", c)
	}
	if c.FirewallMod != "nftables" || c.CNIPRoute != "OFF" || c.IPv6Redir != "ON" {
		t.Fatalf("unexpected firewall config fields: %+v", c)
	}
	if c.DisOverride != "1" {
		t.Fatalf("unexpected disoverride field: %+v", c)
	}
	if c.DBPort != "10090" || c.Secret != "s3" {
		t.Fatalf("unexpected controller config: %+v", c)
	}
	if c.StartDelaySec != 12 || c.BotTGService != "ON" {
		t.Fatalf("unexpected delay/bot config: %+v", c)
	}
	if c.TmpDir != "/tmp/custom" {
		t.Fatalf("expected tmpdir override, got %q", c.TmpDir)
	}
	if c.BinDir != "/opt/shell" {
		t.Fatalf("expected bindir override, got %q", c.BinDir)
	}
	if !strings.Contains(c.Command, "CrashCore run") {
		t.Fatalf("unexpected command: %q", c.Command)
	}
}

func TestLoadConfigFallbackCommandByCore(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	if c.TmpDir != "/tmp/ShellCrash" {
		t.Fatalf("unexpected default tmpdir: %q", c.TmpDir)
	}
	if c.BinDir != crashDir {
		t.Fatalf("expected default bindir to crashdir, got %q", c.BinDir)
	}
	if !strings.Contains(c.Command, "-f $TMPDIR/config.yaml") {
		t.Fatalf("expected clash/meta command default, got %q", c.Command)
	}
	if c.DBPort != "9999" {
		t.Fatalf("expected default db_port, got %q", c.DBPort)
	}
	if c.FirewallMod != "iptables" || c.CNIPRoute != "ON" || c.IPv6Redir != "OFF" {
		t.Fatalf("unexpected defaults for firewall config fields: %+v", c)
	}
	if c.HostsOpt != "ON" {
		t.Fatalf("expected default hosts_opt=ON, got %q", c.HostsOpt)
	}
	if c.SkipCert != "ON" {
		t.Fatalf("expected default skip_cert=ON, got %q", c.SkipCert)
	}
}

func TestLoadConfigSkipCertOverride(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("skip_cert=OFF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	if c.SkipCert != "OFF" {
		t.Fatalf("expected skip_cert override to OFF, got %q", c.SkipCert)
	}
}

func TestExpandCommand(t *testing.T) {
	ctl := Controller{Cfg: Config{CrashDir: "/etc/ShellCrash", BinDir: "/opt/ShellCrash", TmpDir: "/tmp/ShellCrash", Command: "$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"}}
	got := ctl.expandCommand()
	if got != "/tmp/ShellCrash/CrashCore -d /opt/ShellCrash -f /tmp/ShellCrash/config.yaml" {
		t.Fatalf("unexpected command expansion: %q", got)
	}
}
