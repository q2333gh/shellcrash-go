package settingsctl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMenuTogglesSkipCertAndDNSMode(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, "ShellCrash.cfg")
	if err := os.WriteFile(cfgPath, []byte("skip_cert=ON\ndns_mod=redir_host\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("3\n2\n1\n0\n0\n")
	var out bytes.Buffer
	if err := RunMenu(Options{CrashDir: td}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "skip_cert=OFF") {
		t.Fatalf("expected skip_cert OFF, got: %s", text)
	}
	if !strings.Contains(text, "dns_mod=mix") {
		t.Fatalf("expected dns_mod mix, got: %s", text)
	}
}

func TestRunAdvancedPortMenuUpdatesPortsAndAuth(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, "ShellCrash.cfg")
	if err := os.WriteFile(cfgPath, []byte("mix_port=7890\nredir_port=7892\ndns_port=1053\ndb_port=9999\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("1\n8888\n2\nuser:pass\n9\n200\n0\n")
	var out bytes.Buffer
	if err := RunAdvancedPortMenu(Options{CrashDir: td}, in, &out); err != nil {
		t.Fatalf("run adv menu: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(data)
	for _, expect := range []string{"mix_port=8888", "authentication='user:pass'", "table=200"} {
		if !strings.Contains(text, expect) {
			t.Fatalf("expected %q in cfg, got: %s", expect, text)
		}
	}
}

func TestRunIPv6MenuTogglesIPv6(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, "ShellCrash.cfg")
	if err := os.WriteFile(cfgPath, []byte("ipv6_redir=OFF\nipv6_dns=ON\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("1\n2\n0\n")
	var out bytes.Buffer
	if err := RunIPv6Menu(Options{CrashDir: td}, in, &out); err != nil {
		t.Fatalf("run ipv6 menu: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(data)
	for _, expect := range []string{"ipv6_redir=ON", "ipv6_support=ON", "ipv6_dns=OFF"} {
		if !strings.Contains(text, expect) {
			t.Fatalf("expected %q in cfg, got: %s", expect, text)
		}
	}
}

func TestRunMenuDNSAdvancedAndFakeIPFilter(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, "ShellCrash.cfg")
	if err := os.WriteFile(cfgPath, []byte("crashcore=meta\ndns_mod=mix\ndns_port=1053\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader(
		"2\n" + // dns menu
			"9\n" + // dns advanced
			"1\n8.8.8.8|1.1.1.1\n" + // direct dns
			"4\n" + // auto encrypt
			"0\n" + // back to dns menu
			"8\n" + // fake-ip filter
			"example.com\n" +
			"*.example.org\n" +
			"1\n" + // remove first entry
			"0\n" + // back
			"0\n" + // back to root
			"0\n",
	)
	var out bytes.Buffer
	if err := RunMenu(Options{CrashDir: td}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "dns_nameserver='https://dns.alidns.com/dns-query, https://doh.pub/dns-query'") {
		t.Fatalf("expected encrypted dns_nameserver, got: %s", text)
	}
	if !strings.Contains(text, "dns_fallback='https://cloudflare-dns.com/dns-query, https://dns.google/dns-query, https://doh.opendns.com/dns-query'") {
		t.Fatalf("expected encrypted dns_fallback, got: %s", text)
	}

	filterPath := filepath.Join(cfgDir, "fake_ip_filter")
	filterData, err := os.ReadFile(filterPath)
	if err != nil {
		t.Fatalf("read fake_ip_filter: %v", err)
	}
	if got := strings.TrimSpace(string(filterData)); got != "*.example.org" {
		t.Fatalf("unexpected fake_ip_filter content: %q", got)
	}
}

func TestRunDNSAdvancedResetClearsDNSValues(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := filepath.Join(cfgDir, "ShellCrash.cfg")
	if err := os.WriteFile(cfgPath, []byte("dns_nameserver='1.1.1.1'\ndns_fallback='8.8.8.8'\ndns_resolver='223.5.5.5'\ndns_mod=route\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("2\n9\n9\n0\n0\n")
	var out bytes.Buffer
	if err := RunMenu(Options{CrashDir: td}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(data)
	for _, key := range []string{"dns_nameserver=", "dns_fallback=", "dns_resolver="} {
		if strings.Contains(text, key) {
			t.Fatalf("expected %s to be removed, got: %s", key, text)
		}
	}
}
