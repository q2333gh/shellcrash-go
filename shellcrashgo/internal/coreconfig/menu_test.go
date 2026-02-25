package coreconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMenuAddProviderAndClear(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader("a\n1\nmy1\n2\n./providers/a.yaml\na\n0\nd\n1\n0\n")
	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := os.MkdirAll(filepath.Join(root, "providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "providers", "a.yaml"), []byte("proxies: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := RunMenu(MenuOptions{CrashDir: root, TmpDir: filepath.Join(root, "tmp"), In: input, Out: &out, Err: &errOut})
	if err != nil {
		t.Fatalf("run menu error: %v, stderr=%s", err, errOut.String())
	}
	if _, statErr := os.Stat(filepath.Join(root, "configs", "providers.cfg")); !os.IsNotExist(statErr) {
		t.Fatalf("expected providers cfg removed after clear, statErr=%v", statErr)
	}
}

func TestSaveProviderWritesURIFile(t *testing.T) {
	root := t.TempDir()
	p := Provider{Name: "demo", LinkURI: "vmess://abc"}
	if err := saveProvider(root, "", p); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "configs", "providers_uri.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "demo vmess://abc" {
		t.Fatalf("unexpected providers_uri.cfg: %q", string(data))
	}
}

func TestSetProviderAsSourceWritesURL(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := menuState{crashDir: root}
	if err := s.setProviderAsSource(Provider{Name: "n1", Link: "https://example/sub"}, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "Url='https://example/sub'") {
		t.Fatalf("expected Url set, got: %s", text)
	}
	if strings.Contains(text, "Https=") {
		t.Fatalf("expected Https cleared, got: %s", text)
	}
}
