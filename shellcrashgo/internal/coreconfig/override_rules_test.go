package coreconfig

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunOverrideRulesMenuAddRule(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\ndb_port=9999\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	oldHTTP := overrideRulesHTTPDo
	defer func() { overrideRulesHTTPDo = oldHTTP }()
	overrideRulesHTTPDo = func(req *http.Request) (*http.Response, error) {
		body := `{"proxies":{"Auto":{"type":"URLTest"},"X":{"type":"Direct"}}}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	in := strings.NewReader("1\n3\n1.1.1.0/24\n1\n0\n")
	if err := RunOverrideRulesMenu(MenuOptions{CrashDir: root, In: in, Out: io.Discard}); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "yamls", "rules.yaml"))
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}
	want := "- IP-CIDR,1.1.1.0/24,DIRECT,no-resolve\n"
	if string(got) != want {
		t.Fatalf("unexpected rules content:\n%s", string(got))
	}
}

func TestRunOverrideRulesMenuDeleteAndClear(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "yamls"), 0o755); err != nil {
		t.Fatalf("mkdir yamls: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	rules := "# keep\n- DOMAIN,a.com,DIRECT\n- DOMAIN,b.com,REJECT\n"
	if err := os.WriteFile(filepath.Join(root, "yamls", "rules.yaml"), []byte(rules), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}

	in := strings.NewReader("2\n2\n3\n1\n0\n")
	if err := RunOverrideRulesMenu(MenuOptions{CrashDir: root, In: in, Out: io.Discard}); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "yamls", "rules.yaml"))
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}
	if string(got) != "" {
		t.Fatalf("unexpected rules content:\n%s", string(got))
	}
}

func TestRunOverrideRulesMenuToggleBypass(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\nproxies_bypass=OFF\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("4\n1\n0\n")
	if err := RunOverrideRulesMenu(MenuOptions{CrashDir: root, In: in, Out: io.Discard}); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if !strings.Contains(string(cfg), "proxies_bypass=ON\n") {
		t.Fatalf("expected proxies_bypass=ON in cfg:\n%s", string(cfg))
	}
}
