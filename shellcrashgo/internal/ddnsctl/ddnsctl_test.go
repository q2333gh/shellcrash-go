package ddnsctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListServiceIDs(t *testing.T) {
	td := t.TempDir()
	cfg := filepath.Join(td, "ddns")
	content := `
config service 'dynv6'
	option enabled '1'

config service "he_net"
	option enabled '0'
`
	if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := listServiceIDs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "dynv6" || got[1] != "he_net" {
		t.Fatalf("unexpected ids: %#v", got)
	}
}

func TestAddServiceAppendsConfigAndStartsUpdater(t *testing.T) {
	td := t.TempDir()
	cfg := filepath.Join(td, "ddns")
	if err := os.WriteFile(cfg, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	var cmds [][]string
	deps := Deps{
		RunCommand: func(name string, args ...string) ([]byte, error) {
			cmds = append(cmds, append([]string{name}, args...))
			return nil, nil
		},
	}
	err := AddService(Options{ConfigPath: cfg, UpdaterPath: "/usr/lib/ddns/dynamic_dns_updater.sh"}, deps, AddParams{
		ServiceID:     "dynv6",
		ServiceName:   "dynv6.com",
		Domain:        "example.com",
		Username:      "u",
		Password:      "p",
		CheckInterval: 9,
		ForceInterval: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"config service 'dynv6'",
		"option service_name 'dynv6.com'",
		"option domain 'example.com'",
		"option username 'u'",
		"option password 'p'",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in config:\n%s", want, text)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("expected one updater call, got %d", len(cmds))
	}
	if got := strings.Join(cmds[0], " "); got != "/usr/lib/ddns/dynamic_dns_updater.sh -S dynv6 start" {
		t.Fatalf("unexpected updater command: %s", got)
	}
}

func TestReadLastRegisteredIP(t *testing.T) {
	td := t.TempDir()
	logFile := filepath.Join(td, "x.log")
	content := `
[foo] Registered IP '1.1.1.1'
[bar] Registered IP '8.8.8.8'
`
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readLastRegisteredIP(logFile)
	if got != "8.8.8.8" {
		t.Fatalf("unexpected ip: %q", got)
	}
}
