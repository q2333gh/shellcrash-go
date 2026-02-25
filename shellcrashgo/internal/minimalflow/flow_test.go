package minimalflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultMetaSubconvertOption1Hy2Flow(t *testing.T) {
	td := t.TempDir()
	crashdir := filepath.Join(td, "ShellCrash")

	mustMkdirAll(t, filepath.Join(crashdir, "configs"))
	mustMkdirAll(t, filepath.Join(crashdir, "yamls"))
	mustMkdirAll(t, filepath.Join(crashdir, "jsons"))

	mustWriteFile(t, filepath.Join(crashdir, "configs", "providers.cfg"), "sub1 https://sub.example.com/a.yaml 3 12 clash.meta ##\n")
	mustWriteFile(t, filepath.Join(crashdir, "configs", "providers_uri.cfg"), "hy2a hy2://node-a\nhy2b hy2://node-b\n")

	subPath := filepath.Join(td, "sub")
	mustWriteFile(t, subPath,
		"proxies:\n"+
			"  - {name: hy2-a, type: hysteria2, server: a.example.com, port: 443}\n"+
			"  - {name: hy2-b, type: hysteria2, server: b.example.com, port: 443}\n"+
			"rules:\n"+
			"  - MATCH,DIRECT\n",
	)

	mustWriteFile(t, filepath.Join(crashdir, "configs", "servers.list"),
		"401 test file://"+td+" ua\n"+
			"501 rule https://rules.example.com/template.ini\n",
	)

	err := Run(Options{
		CrashDir: crashdir,
		TmpDir:   filepath.Join(td, "tmp"),
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	cfgText := mustReadFile(t, filepath.Join(crashdir, "configs", "ShellCrash.cfg"))
	outYAML := mustReadFile(t, filepath.Join(crashdir, "yamls", "config.yaml"))

	if !strings.Contains(cfgText, "crashcore=meta") {
		t.Fatalf("expected crashcore=meta in cfg, got:\n%s", cfgText)
	}
	if !strings.Contains(cfgText, "Https='file://"+td+"/sub?") {
		t.Fatalf("expected Https file:// sub link in cfg, got:\n%s", cfgText)
	}
	if !strings.Contains(outYAML, "type: hysteria2") {
		t.Fatalf("expected hy2 node in yaml, got:\n%s", outYAML)
	}
}

func mustWriteFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
