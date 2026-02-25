package startctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCorePreChecksDownloadsClashGeoAssets(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	yamlsDir := filepath.Join(crashDir, "yamls")

	for _, p := range []string{cfgDir, binDir, tmpDir, yamlsDir} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfgData := "crashcore=clash\ndns_mod=mix\nfirewall_mod=nftables\ncn_ip_route=OFF\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+binDir+"\nTMPDIR="+tmpDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	coreConfig := filepath.Join(yamlsDir, "config.yaml")
	coreContent := "rules:\n - GEOIP,CN,DIRECT\n - GEOSITE,CN,DIRECT\n"
	if err := os.WriteFile(coreConfig, []byte(coreContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "CrashCore"), []byte(strings.Repeat("x", 3001)), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}

	var downloaded []string
	startctlGeoDownloader = func(dst string, remoteName string) error {
		downloaded = append(downloaded, remoteName)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, []byte(strings.Repeat("d", 64)), 0o644)
	}
	t.Cleanup(func() {
		startctlGeoDownloader = nil
	})

	if err := (&ctl).runCorePreChecks(coreConfig); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cn_mini.mmdb", "geosite.dat", "mrs_geosite_cn.mrs"} {
		found := false
		for _, got := range downloaded {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing download %q, got: %v", want, downloaded)
		}
	}
}

func TestRestoreIPSetFromCIDRFileBuildsRestoreFile(t *testing.T) {
	td := t.TempDir()
	cidrFile := filepath.Join(td, "cn_ip.txt")
	if err := os.WriteFile(cidrFile, []byte("1.1.1.0/24\n#comment\n\n2.2.2.0/24\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	destroyCalls := 0
	var restoreText string
	err := restoreIPSetFromCIDRFile(
		cidrFile,
		"cn_ip",
		"inet",
		"10240",
		func(name string) {
			if name == "cn_ip" {
				destroyCalls++
			}
		},
		func(path string) error {
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			restoreText = string(b)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if destroyCalls != 1 {
		t.Fatalf("expected destroy to be called once, got %d", destroyCalls)
	}
	if !strings.Contains(restoreText, "create cn_ip hash:net family inet hashsize 10240 maxelem 10240") {
		t.Fatalf("missing create command: %s", restoreText)
	}
	if !strings.Contains(restoreText, "add cn_ip 1.1.1.0/24") || !strings.Contains(restoreText, "add cn_ip 2.2.2.0/24") {
		t.Fatalf("missing add commands: %s", restoreText)
	}
}
