package startctl

import (
	"compress/gzip"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCorePreChecksSwitchesToMetaForVLESS(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=clash\nredir_mod=Redir\nfirewall_area=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+binDir+"\nTMPDIR="+tmpDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(crashDir, "yamls", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coreConfig, []byte("proxies:\n- { name: a, type: vless, server: x }\n"), 0o644); err != nil {
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
	if err := (&ctl).runCorePreChecks(coreConfig); err != nil {
		t.Fatal(err)
	}
	if ctl.Cfg.CrashCore != "meta" {
		t.Fatalf("expected crashcore switched to meta, got %q", ctl.Cfg.CrashCore)
	}
	updatedCfg, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedCfg), "crashcore=meta") {
		t.Fatalf("cfg not updated: %s", string(updatedCfg))
	}
}

func TestRunCorePreChecksUnpacksGzipCore(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\nfirewall_area=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+binDir+"\nTMPDIR="+tmpDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(crashDir, "yamls", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coreConfig, []byte("proxies:\n- { name: a, type: socks5, server: x }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gzPath := filepath.Join(binDir, "CrashCore.gz")
	gzw, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(gzw)
	payload := make([]byte, 6000)
	for i := range payload {
		payload[i] = byte(rand.Intn(255) + 1)
	}
	if _, err := zw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	if err := (&ctl).runCorePreChecks(coreConfig); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(tmpDir, "CrashCore"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 2000 {
		t.Fatalf("unexpected CrashCore size: %d", info.Size())
	}
}

func TestRunCorePreChecksBuildsClashRuntimeConfig(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\nfirewall_area=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+binDir+"\nTMPDIR="+tmpDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(crashDir, "yamls", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
		t.Fatal(err)
	}
	configText := "mixed-port: 7890\nproxies: []\n"
	if err := os.WriteFile(coreConfig, []byte(configText), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "CrashCore"), []byte(strings.Repeat("x", 3001)), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	if err := (&ctl).runCorePreChecks(coreConfig); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "external-controller: :9999") ||
		!strings.Contains(text, "dns:") ||
		!strings.Contains(text, "proxies: []") {
		t.Fatalf("unexpected tmp config content: %q", text)
	}
}

func TestRunCorePreChecksBuildsSingboxRuntimeConfigWithCustomJSON(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	jsonDir := filepath.Join(crashDir, "jsons")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=singboxr\ndisoverride=0\nfirewall_area=1\nmix_port=8000\ndns_port=5300\nredir_port=7898\ntproxy_port=7899\nauthentication=user:pass\ndns_mod=redir_host\ncn_ip_route=OFF\nfwmark=100\nskip_cert=ON\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+binDir+"\nTMPDIR="+tmpDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(jsonDir, "config.json")
	if err := os.WriteFile(coreConfig, []byte("{\"log\":{\"level\":\"info\"},\"outbounds\":[{\"tag\":\"node1\",\"type\":\"socks\",\"server\":\"1.1.1.1\",\"server_port\":443,\"tls\":{\"enabled\":true,\"insecure\":false}}]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jsonDir, "dns.json"), []byte("{\"dns\":{\"servers\":[]}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "CrashCore"), []byte(strings.Repeat("x", 3001)), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	if err := (&ctl).runCorePreChecks(coreConfig); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "jsons", "00_config.json")); err != nil {
		t.Fatalf("missing base runtime config: %v", err)
	}
	custom, err := os.ReadFile(filepath.Join(tmpDir, "jsons", "50_dns.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(custom), "\"dns\"") {
		t.Fatalf("custom json not copied: %s", string(custom))
	}
	inbounds, err := os.ReadFile(filepath.Join(tmpDir, "jsons", "30_inbounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(inbounds), "\"listen_port\": 8000") ||
		!strings.Contains(string(inbounds), "\"username\": \"user\"") {
		t.Fatalf("generated inbounds json missing expected values: %s", string(inbounds))
	}
	dns, err := os.ReadFile(filepath.Join(tmpDir, "jsons", "40_dns.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dns), "\"server_port\": 53") ||
		!strings.Contains(string(dns), "\"final\": \"dns_proxy\"") {
		t.Fatalf("generated dns json missing expected values: %s", string(dns))
	}
	baseOut, err := os.ReadFile(filepath.Join(tmpDir, "jsons", "00_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(baseOut), "\"insecure\": true") {
		t.Fatalf("expected skip_cert=ON to force insecure=true in runtime outbounds, got: %s", string(baseOut))
	}
}

func TestRunCorePreChecksRejectsClashConfigWithoutNodesOrProviders(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\nfirewall_area=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+filepath.Join(td, "bin")+"\nTMPDIR="+filepath.Join(td, "tmp")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(crashDir, "yamls", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coreConfig, []byte("proxies:\n  - name: no-server-field\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	err = (&ctl).runCorePreChecks(coreConfig)
	if err == nil || !strings.Contains(err.Error(), "no valid clash nodes/providers") {
		t.Fatalf("expected clash node/provider validation error, got %v", err)
	}
}

func TestRunCorePreChecksRejectsClashLegacyFormat(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\nfirewall_area=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+filepath.Join(td, "bin")+"\nTMPDIR="+filepath.Join(td, "tmp")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(crashDir, "yamls", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coreConfig, []byte("proxy-providers:\n  p: {type: http, url: http://x}\nProxy Group: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	err = (&ctl).runCorePreChecks(coreConfig)
	if err == nil || !strings.Contains(err.Error(), "unsupported legacy clash format") {
		t.Fatalf("expected clash legacy-format validation error, got %v", err)
	}
}

func TestRunCorePreChecksRejectsLegacySingboxSNI(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	jsonDir := filepath.Join(crashDir, "jsons")
	for _, p := range []string{cfgDir, binDir, tmpDir, jsonDir} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=singboxr\nfirewall_area=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+binDir+"\nTMPDIR="+tmpDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	coreConfig := filepath.Join(jsonDir, "config.json")
	if err := os.WriteFile(coreConfig, []byte("{\"outbounds\":[{\"type\":\"socks\",\"server\":\"1.1.1.1\",\"server_port\":443,\"sni\":\"old.example\"}]}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	err = (&ctl).runCorePreChecks(coreConfig)
	if err == nil || !strings.Contains(err.Error(), "legacy sing-box (<1.12) format") {
		t.Fatalf("expected sing-box legacy-format validation error, got %v", err)
	}
}
