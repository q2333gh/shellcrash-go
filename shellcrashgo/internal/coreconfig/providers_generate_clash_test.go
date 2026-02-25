package coreconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProvidersGenerateClashFromConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "rules", "clash_providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\nskip_cert=ON\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "providers.cfg"), []byte("a https://example.com/sub 5 10 uaX #ex #in\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl := "proxy-groups:\n  - {name: Auto, type: select, use: [{providers_tags}]}\nrules:\n  - MATCH,DIRECT\n"
	if err := os.WriteFile(filepath.Join(root, "rules", "clash_providers", "clash-default.yaml"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "clash_providers.list"), []byte("默认 clash-default.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunProvidersGenerateClash(Options{CrashDir: root, TmpDir: filepath.Join(root, "tmp")}, nil); err != nil {
		t.Fatalf("run generate clash: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "yamls", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	checks := []string{
		"proxy-providers:",
		"a:",
		"interval: 36000",
		"interval: 300",
		"skip-cert-verify: true",
		"use: [a]",
		"exclude-filter: \"ex\"",
		"rules:",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("expected generated config contains %q, got:\n%s", want, text)
		}
	}
}

func TestRunProvidersGenerateClashFromArgs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "rules", "clash_providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "clash_providers.list"), []byte("默认 t.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl := "proxy-groups:\n  - {name: Auto, type: select, use: [{providers_tags}]}\nrules:\n  - MATCH,DIRECT\n"
	if err := os.WriteFile(filepath.Join(root, "rules", "clash_providers", "t.yaml"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{"one", "./providers/one.yaml", "3", "12", "ua", "#x", "#y"}
	if err := RunProvidersGenerateClash(Options{CrashDir: root, TmpDir: filepath.Join(root, "tmp")}, args); err != nil {
		t.Fatalf("run generate clash by args: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "yamls", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "type: file") || !strings.Contains(text, "path: \"./providers/one.yaml\"") {
		t.Fatalf("expected file provider output, got:\n%s", text)
	}
	if !strings.Contains(text, "use: [one]") {
		t.Fatalf("expected provider tag replacement, got:\n%s", text)
	}
}
