package coreconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunClashViaURL(t *testing.T) {
	root := t.TempDir()
	tmp := filepath.Join(root, "tmp")
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	srvRoot := filepath.Join(root, "srv")
	if err := os.MkdirAll(srvRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srvRoot, "sub"), []byte("proxies:\n  - {name: n1, type: ss, server: 1.1.1.1, port: 443}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := "crashcore=meta\nUrl='ss://example'\nuser_agent=auto\nserver_link=1\nrule_link=1\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	servers := "401 s file://" + srvRoot + " ua\n501 r https://rules.example/config.ini\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "servers.list"), []byte(servers), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Run(Options{CrashDir: root, TmpDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if res.Target != "clash" || res.Format != "yaml" {
		t.Fatalf("unexpected result target/format: %#v", res)
	}

	out, err := os.ReadFile(filepath.Join(root, "yamls", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "proxies:") {
		t.Fatalf("unexpected output config: %s", out)
	}
	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cfgOut), "Https='") {
		t.Fatalf("expected Https cleared, got:\n%s", cfgOut)
	}
	if !strings.Contains(string(cfgOut), "user_agent=clash.meta/mihomo") {
		t.Fatalf("expected user_agent persisted, got:\n%s", cfgOut)
	}
}

func TestRunRetryServerSwitch(t *testing.T) {
	root := t.TempDir()
	tmp := filepath.Join(root, "tmp")
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\nUrl='vmess://abc'\nserver_link=1\nrule_link=1\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	goodRoot := filepath.Join(root, "good")
	if err := os.MkdirAll(goodRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goodRoot, "sub"), []byte("proxy-providers:\n  p1:\n    type: http\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	servers := "401 bad file:///not-found-12345 ua\n402 ok file://" + goodRoot + " ua\n501 rule https://rules.example/config.ini\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "servers.list"), []byte(servers), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(Options{CrashDir: root, TmpDir: tmp}); err != nil {
		t.Fatal(err)
	}
	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgOut), "server_link=2") {
		t.Fatalf("expected server_link switch to 2, got:\n%s", cfgOut)
	}
}

func TestRunSingboxLegacyRejected(t *testing.T) {
	root := t.TempDir()
	tmp := filepath.Join(root, "tmp")
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "servers.list"), []byte("401 s https://x ua\n501 r https://x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bad := filepath.Join(root, "legacy.json")
	if err := os.WriteFile(bad, []byte(`{"outbounds":[{"type":"vmess","sni":"old"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=singbox\nHttps='file://" + bad + "'\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Run(Options{CrashDir: root, TmpDir: tmp})
	if err == nil {
		t.Fatal("expected legacy singbox rejection")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSanitizeSingboxConfigRemovesDirectURLTests(t *testing.T) {
	raw := []byte(`{"outbounds":[{"type":"urltest","tag":"AUTO","outbounds":["DIRECT"]},{"type":"selector","tag":"GLOBAL","outbounds":["AUTO","DIRECT"]},{"type":"urltest","tag":"KEEP","outbounds":["PROXY"]}],"route":{"rules":[{"outbound":"AUTO"}]}}`)
	out := string(sanitizeSingboxConfig(raw, true))
	if strings.Contains(out, `"tag":"AUTO"`) {
		t.Fatalf("expected AUTO group removed, got: %s", out)
	}
	if strings.Contains(out, `"outbound":"AUTO"`) {
		t.Fatalf("expected AUTO references removed, got: %s", out)
	}
	if !strings.Contains(out, `"tag":"KEEP"`) {
		t.Fatalf("expected non-direct urltest group preserved, got: %s", out)
	}
}

func TestSanitizeSingboxConfigNormalizesCompactLegacy(t *testing.T) {
	raw := []byte(`"inbounds":[{"type":"mixed"}],{"tag":"dns-out","type":"dns"}]`)
	out := string(sanitizeSingboxConfig(raw, false))
	if !strings.HasPrefix(out, `{"inbounds":`) {
		t.Fatalf("expected inbounds root normalization, got: %s", out)
	}
	if strings.Contains(out, `"dns-out"`) {
		t.Fatalf("expected dns-out fragment removed, got: %s", out)
	}
}
