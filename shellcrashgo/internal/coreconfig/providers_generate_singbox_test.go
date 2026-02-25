package coreconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProvidersGenerateSingboxFromConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "rules", "singbox_providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := "crashcore=singboxr\nskip_cert=ON\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "providers.cfg"), []byte("a https://example.com/sub 5 10 uaX #ex #in\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "singbox_providers.list"), []byte("默认 t.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl := "//comment\n{\n  \"outbounds\": [\n    {\"tag\":\"GROUP\",\"type\":\"selector\",\"outbounds\":[{providers_tags},\"DIRECT\"]}\n  ],\n  \"route\": {\"final\":\"GROUP\"}\n}\n"
	if err := os.WriteFile(filepath.Join(root, "rules", "singbox_providers", "t.json"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	base := "{\"dns\":{\"servers\":[\"8.8.8.8\"]}}"
	if err := os.WriteFile(filepath.Join(root, "tmp", "config.json"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	err := RunProvidersGenerateSingbox(Options{
		CrashDir: root,
		TmpDir:   filepath.Join(root, "tmp"),
	}, nil)
	if err != nil {
		t.Fatalf("run providers generate singbox: %v", err)
	}

	outPath := filepath.Join(root, "jsons", "config.json")
	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	text := string(out)
	checks := []string{
		`"providers"`,
		`"tag": "a"`,
		`"type": "remote"`,
		`"user_agent": "uaX"`,
		`"update_interval": "10h"`,
		`"exclude": "ex"`,
		`"include": "in"`,
		`"insecure": true`,
		`"tolerance": 100`,
		`"dns"`,
		`"route"`,
	}
	for _, c := range checks {
		if !strings.Contains(text, c) {
			t.Fatalf("expected output contains %q, got: %s", c, text)
		}
	}
}

func TestRunProvidersGenerateSingboxFromArgsLocal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "rules", "singbox_providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=singboxr\nskip_cert=OFF\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "singbox_providers.list"), []byte("默认 t.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl := "{\"outbounds\":[{\"tag\":\"GROUP\",\"type\":\"selector\",\"outbounds\":[{providers_tags}]}],\"route\":{\"final\":\"GROUP\"}}"
	if err := os.WriteFile(filepath.Join(root, "rules", "singbox_providers", "t.json"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{"one", "./providers/one.yaml", "3", "12", "ua", "#x", "#y"}
	if err := RunProvidersGenerateSingbox(Options{CrashDir: root, TmpDir: filepath.Join(root, "tmp")}, args); err != nil {
		t.Fatalf("run providers generate singbox by args: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(root, "jsons", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, `"type": "local"`) || !strings.Contains(text, `"path": "./providers/one.yaml"`) {
		t.Fatalf("expected local provider fields, got: %s", text)
	}
	if !strings.Contains(text, `"insecure": false`) {
		t.Fatalf("expected skip_cert=OFF keeps insecure false, got: %s", text)
	}
}
