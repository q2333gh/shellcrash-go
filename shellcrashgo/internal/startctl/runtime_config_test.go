package startctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareClashRuntimeConfigDisOverride(t *testing.T) {
	td := t.TempDir()
	core := filepath.Join(td, "base.yaml")
	base := "mixed-port: 1000\n"
	if err := os.WriteFile(core, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		TmpDir:      filepath.Join(td, "tmp"),
		BinDir:      filepath.Join(td, "bin"),
		CrashCore:   "meta",
		DisOverride: "1",
	}}
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(ctl.Cfg.TmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != base {
		t.Fatalf("expected unchanged config when disoverride=1, got:\n%s", string(got))
	}
}

func TestPrepareClashRuntimeConfigAppliesOverlay(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "crash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "configs", "fake_ip_filter.list"), []byte("example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	core := filepath.Join(crashDir, "core.yaml")
	if err := os.WriteFile(core, []byte("proxies: []\nrules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		CrashDir:      crashDir,
		TmpDir:        filepath.Join(td, "tmp"),
		BinDir:        filepath.Join(td, "bin"),
		CrashCore:     "meta",
		DisOverride:   "0",
		RedirMod:      "Mix",
		DNSMod:        "mix",
		DBPort:        "9999",
		MixPort:       "7890",
		RedirPort:     "7892",
		TProxyPort:    "7893",
		DNSPort:       "1053",
		DNSNameServer: "223.5.5.5, 1.2.4.8",
		DNSFallback:   "1.1.1.1, 8.8.8.8",
		DNSResolver:   "223.5.5.5",
		RoutingMark:   "7894",
		Sniffer:       "ON",
		IPv6DNS:       "ON",
	}}
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(ctl.Cfg.TmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"mixed-port: 7890",
		"external-controller: :9999",
		"tun: {enable: true, stack: system, device: utun, auto-route: false, auto-detect-interface: false}",
		"sniffer: {enable: true",
		"dns:",
		"rule-providers:",
		"cn: {type: http, behavior: domain, format: mrs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, text)
		}
	}
}

func TestPrepareClashRuntimeConfigMergesUserAndCustomSections(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "crash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "yamls"), 0o755); err != nil {
		t.Fatal(err)
	}

	core := filepath.Join(crashDir, "core.yaml")
	coreText := "proxies:\n  - {name: base, type: ss, server: 1.1.1.1, port: 443, cipher: aes-128-gcm, password: x}\nproxy-groups:\n  - {name: P, type: select, proxies: [base]}\nrules:\n  - MATCH,DIRECT\n"
	if err := os.WriteFile(core, []byte(coreText), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "yamls", "user.yaml"), []byte("mode: Global\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "yamls", "proxies.yaml"), []byte("- {name: custom, type: ss, server: 2.2.2.2, port: 443, cipher: aes-128-gcm, password: y}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "yamls", "rules.yaml"), []byte("- DOMAIN,example.com,DIRECT\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		CrashDir:      crashDir,
		TmpDir:        filepath.Join(td, "tmp"),
		BinDir:        filepath.Join(td, "bin"),
		CrashCore:     "meta",
		DisOverride:   "0",
		RedirMod:      "Redir",
		DNSMod:        "fake-ip",
		DBPort:        "9999",
		MixPort:       "7890",
		RedirPort:     "7892",
		TProxyPort:    "7893",
		DNSPort:       "1053",
		DNSNameServer: "223.5.5.5",
		DNSFallback:   "1.1.1.1",
		DNSResolver:   "223.5.5.5",
		RoutingMark:   "7894",
	}}
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(filepath.Join(ctl.Cfg.TmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "mode: Global") {
		t.Fatalf("expected mode override from user.yaml, got:\n%s", text)
	}
	if strings.Contains(text, "mode: Rule") {
		t.Fatalf("expected base mode to be removed when user.yaml overrides it, got:\n%s", text)
	}
	if !strings.Contains(text, "name: custom") {
		t.Fatalf("expected custom proxies merge, got:\n%s", text)
	}
	if !strings.Contains(text, "DOMAIN,example.com,DIRECT") {
		t.Fatalf("expected custom rules merge, got:\n%s", text)
	}
}

func TestPrepareClashRuntimeConfigValidationFallback(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "crash")
	if err := os.MkdirAll(filepath.Join(crashDir, "yamls"), 0o755); err != nil {
		t.Fatal(err)
	}
	core := filepath.Join(crashDir, "core.yaml")
	if err := os.WriteFile(core, []byte("proxies: []\nrules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "yamls", "user.yaml"), []byte("mode: Global\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	coreBin := filepath.Join(tmpDir, "CrashCore")
	script := "#!/bin/sh\nexit 1\n"
	script += strings.Repeat("#", 3000)
	if err := os.WriteFile(coreBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		CrashDir:    crashDir,
		TmpDir:      tmpDir,
		BinDir:      filepath.Join(td, "bin"),
		CrashCore:   "meta",
		DisOverride: "0",
		DNSMod:      "fake-ip",
	}}
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "mode: Global") {
		t.Fatalf("expected fallback config without custom merge when validation fails, got:\n%s", text)
	}
	if !strings.Contains(text, "mode: Rule") {
		t.Fatalf("expected fallback baseline set values, got:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "error.yaml")); err != nil {
		t.Fatalf("expected error.yaml to be written on failed validation: %v", err)
	}
}

func TestPrepareClashRuntimeConfigHostsOverlay(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "crash")
	if err := os.MkdirAll(filepath.Join(crashDir, "yamls"), 0o755); err != nil {
		t.Fatal(err)
	}
	core := filepath.Join(crashDir, "core.yaml")
	if err := os.WriteFile(core, []byte("proxies: []\nrules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		CrashDir:    crashDir,
		TmpDir:      filepath.Join(td, "tmp"),
		BinDir:      filepath.Join(td, "bin"),
		CrashCore:   "meta",
		DisOverride: "0",
		DNSMod:      "fake-ip",
		HostsOpt:    "ON",
	}}
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(ctl.Cfg.TmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "use-system-hosts: true") || !strings.Contains(text, "'time.android.com': 203.107.6.88") {
		t.Fatalf("expected generated hosts section, got:\n%s", text)
	}
	if !strings.Contains(text, "'services.googleapis.cn': services.googleapis.com") {
		t.Fatalf("expected meta-only services.googleapis.cn host mapping, got:\n%s", text)
	}
}

func TestPrepareClashRuntimeConfigSkipCertToggle(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "crash")
	if err := os.MkdirAll(filepath.Join(crashDir, "yamls"), 0o755); err != nil {
		t.Fatal(err)
	}
	core := filepath.Join(crashDir, "core.yaml")
	coreText := "proxies:\n  - {name: x, type: ss, server: 1.1.1.1, port: 443, skip-cert-verify: false}\nrules: []\n"
	if err := os.WriteFile(core, []byte(coreText), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		CrashDir:    crashDir,
		TmpDir:      filepath.Join(td, "tmp"),
		BinDir:      filepath.Join(td, "bin"),
		CrashCore:   "meta",
		DisOverride: "0",
		DNSMod:      "fake-ip",
		SkipCert:    "OFF",
	}}
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(ctl.Cfg.TmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "skip-cert-verify: false") {
		t.Fatalf("expected skip-cert-verify to stay false when skip_cert=OFF, got:\n%s", string(out))
	}

	ctl.Cfg.SkipCert = "ON"
	if err := ctl.prepareClashRuntimeConfig(core); err != nil {
		t.Fatal(err)
	}
	out, err = os.ReadFile(filepath.Join(ctl.Cfg.TmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "skip-cert-verify: true") {
		t.Fatalf("expected skip-cert-verify true when skip_cert=ON, got:\n%s", string(out))
	}
}
