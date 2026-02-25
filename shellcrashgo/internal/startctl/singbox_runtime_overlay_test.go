package startctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseSBServer(t *testing.T) {
	cases := []struct {
		in   string
		typ  string
		host string
		port int
	}{
		{"1.1.1.1, 8.8.8.8", "udp", "1.1.1.1", 53},
		{"https://dns.google/dns-query", "https", "dns.google", 443},
		{"tls://[2400:3200::1]:853", "tls", "2400:3200::1", 853},
	}
	for _, tc := range cases {
		got := parseSBServer(tc.in)
		if got.Type != tc.typ || got.Host != tc.host || got.Port != tc.port {
			t.Fatalf("parseSBServer(%q) got %+v, want type=%s host=%s port=%d", tc.in, got, tc.typ, tc.host, tc.port)
		}
	}
}

func TestParseAuth(t *testing.T) {
	user, pass, ok := parseAuth("u:p")
	if !ok || user != "u" || pass != "p" {
		t.Fatalf("unexpected parseAuth result: ok=%v user=%q pass=%q", ok, user, pass)
	}
	if _, _, ok := parseAuth("invalid"); ok {
		t.Fatal("expected parseAuth(invalid) to fail")
	}
}

func TestWriteSingboxHosts(t *testing.T) {
	td := t.TempDir()
	ctl := Controller{Cfg: Config{HostsOpt: "ON"}}
	out := filepath.Join(td, "hosts.json")
	if err := ctl.writeSingboxHosts(out); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "\"type\": \"hosts\"") || !strings.Contains(text, "\"time.android.com\": \"203.107.6.88\"") {
		t.Fatalf("unexpected hosts overlay content:\n%s", text)
	}
}

func TestWriteSingboxHostsDisabled(t *testing.T) {
	td := t.TempDir()
	ctl := Controller{Cfg: Config{HostsOpt: "OFF"}}
	out := filepath.Join(td, "hosts.json")
	if err := ctl.writeSingboxHosts(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("expected no hosts overlay file when hosts_opt=OFF, got err=%v", err)
	}
}

func TestBuildSingboxFakeIPFilterRules(t *testing.T) {
	td := t.TempDir()
	if err := os.MkdirAll(filepath.Join(td, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# comment\nexample.com\n*.suffix.test\n+.plus.test\nfoo.*.bar\n+.*.baz\nMijia Cloud\n"
	if err := os.WriteFile(filepath.Join(td, "configs", "fake_ip_filter"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{CrashDir: td}}
	rules := ctl.buildSingboxFakeIPFilterRules()
	if len(rules) != 3 {
		t.Fatalf("expected 3 rule groups, got %d: %#v", len(rules), rules)
	}
	wantDomain := []string{"example.com"}
	gotDomain, _ := rules[0]["domain"].([]string)
	if !reflect.DeepEqual(gotDomain, wantDomain) {
		t.Fatalf("unexpected domain rules: got=%v want=%v", gotDomain, wantDomain)
	}
	wantSuffix := []string{"suffix.test", "plus.test"}
	gotSuffix, _ := rules[1]["domain_suffix"].([]string)
	if !reflect.DeepEqual(gotSuffix, wantSuffix) {
		t.Fatalf("unexpected suffix rules: got=%v want=%v", gotSuffix, wantSuffix)
	}
	wantRegex := []string{`foo\..*\.bar`, `.+\..*\.baz`}
	gotRegex, _ := rules[2]["domain_regex"].([]string)
	if !reflect.DeepEqual(gotRegex, wantRegex) {
		t.Fatalf("unexpected regex rules: got=%v want=%v", gotRegex, wantRegex)
	}
}

func TestWriteSingboxDNSAddsFakeIPRules(t *testing.T) {
	td := t.TempDir()
	if err := os.MkdirAll(filepath.Join(td, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "configs", "fake_ip_filter"), []byte("example.com\n*.suffix.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{Cfg: Config{
		CrashDir:      td,
		DNSMod:        "fake-ip",
		DNSFallback:   "1.1.1.1",
		DNSNameServer: "223.5.5.5",
		DNSResolver:   "223.5.5.5",
		RoutingMark:   "7894",
	}}
	out := filepath.Join(td, "dns.json")
	if err := ctl.writeSingboxDNS(out); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	dnsMap, ok := parsed["dns"].(map[string]any)
	if !ok {
		t.Fatalf("missing dns object: %s", string(raw))
	}
	rules, ok := dnsMap["rules"].([]any)
	if !ok {
		t.Fatalf("missing dns.rules: %s", string(raw))
	}
	text := string(raw)
	if !strings.Contains(text, `"domain_suffix": [`) || !strings.Contains(text, `"services.googleapis.cn"`) {
		t.Fatalf("expected services.googleapis.cn fake-ip helper rule in output:\n%s", text)
	}
	if !strings.Contains(text, `"domain": [`) || !strings.Contains(text, `"example.com"`) {
		t.Fatalf("expected fake-ip domain filter rule in output:\n%s", text)
	}
	if !strings.Contains(text, `"domain_suffix": [`) || !strings.Contains(text, `"suffix.test"`) {
		t.Fatalf("expected fake-ip suffix filter rule in output:\n%s", text)
	}
	if len(rules) < 6 {
		t.Fatalf("expected expanded dns rules list, got %d", len(rules))
	}
}
