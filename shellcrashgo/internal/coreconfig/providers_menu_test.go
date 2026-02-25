package coreconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProvidersMenuSelectTemplateAndCleanup(t *testing.T) {
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
	list := "tmplA a.yaml\ntmplB b.yaml\n"
	if err := os.WriteFile(filepath.Join(root, "rules", "clash_providers", "clash_providers.list"), []byte(list), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "providers", "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("2\n2\n3\n1\n0\n")
	var out bytes.Buffer
	if err := RunProvidersMenu(MenuOptions{
		CrashDir: root,
		TmpDir:   filepath.Join(root, "tmp"),
		In:       input,
		Out:      &out,
		Err:      &out,
	}); err != nil {
		t.Fatalf("run providers menu: %v", err)
	}

	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	cfgText := string(cfgData)
	if !strings.Contains(cfgText, "provider_temp_clash=b.yaml") {
		t.Fatalf("expected provider_temp_clash set to b.yaml, got: %s", cfgText)
	}
	if _, err := os.Stat(filepath.Join(root, "providers")); !os.IsNotExist(err) {
		t.Fatalf("expected providers directory removed, stat err=%v", err)
	}
}

func TestRunProvidersMenuCustomTemplatePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "rules", "clash_providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=clash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	list := "tmplA a.yaml\n"
	if err := os.WriteFile(filepath.Join(root, "rules", "clash_providers", "clash_providers.list"), []byte(list), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(root, "my-template.yaml")
	if err := os.WriteFile(custom, []byte("proxy-groups:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("2\na\n" + custom + "\n0\n")
	var out bytes.Buffer
	if err := RunProvidersMenu(MenuOptions{
		CrashDir: root,
		In:       input,
		Out:      &out,
		Err:      &out,
	}); err != nil {
		t.Fatalf("run providers menu: %v", err)
	}

	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "provider_temp_clash="+custom) {
		t.Fatalf("expected custom template path set, got: %s", string(cfgData))
	}
}
