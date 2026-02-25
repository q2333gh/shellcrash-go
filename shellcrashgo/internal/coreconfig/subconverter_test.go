package coreconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSubconverterURL(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	providers := "a https://example.com/sub\nb ./providers/local.yaml\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "providers.cfg"), []byte(providers), 0o644); err != nil {
		t.Fatal(err)
	}
	uris := "vmess vmess://abc\nss ss://def\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "providers_uri.cfg"), []byte(uris), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := buildSubconverterURL(root)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/sub|vmess://abc|ss://def#ss"
	if got != want {
		t.Fatalf("unexpected combined url: got=%q want=%q", got, want)
	}
}

func TestRunSubconverterGenerateSetsUrlAndCallsRun(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\nHttps='https://old.example/x'\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "providers.cfg"), []byte("a https://example.com/sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := subconverterRun
	defer func() { subconverterRun = orig }()
	called := false
	subconverterRun = func(opts Options) (Result, error) {
		called = true
		if opts.CrashDir != root {
			t.Fatalf("unexpected crash dir: %s", opts.CrashDir)
		}
		return Result{CoreConfig: "ok"}, nil
	}

	if err := RunSubconverterGenerate(Options{CrashDir: root, TmpDir: filepath.Join(root, "tmp")}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected RunSubconverterGenerate to invoke coreconfig Run")
	}
	out, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "Url='https://example.com/sub'") {
		t.Fatalf("expected Url written, got:\n%s", text)
	}
	if strings.Contains(text, "Https='https://old.example/x'") {
		t.Fatalf("expected Https cleared, got:\n%s", text)
	}
}

func TestRunSubconverterExcludeMenuWritesConfig(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunSubconverterExcludeMenu(MenuOptions{
		CrashDir: root,
		In:       strings.NewReader("test|hk\n"),
		Out:      &out,
		Err:      &out,
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "exclude='test|hk'") && !strings.Contains(string(cfg), "exclude=test|hk") {
		t.Fatalf("expected exclude key, got:\n%s", string(cfg))
	}
}
