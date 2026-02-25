package firewall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaultsAndGateway(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgData := "common_ports=OFF\ncn_ip_route=OFF\nredir_port=1234\nfwmark=4444\ntable=200\n"
	cfgData += "cust_host_ipv4=192.168.55.0/24,10.10.0.0/16\nreplace_default_host_ipv4=ON\n"
	cfgData += "reserve_ipv4=1.1.1.0/24\nreserve_ipv6=fd00::/8\n"
	cfgData += "ts_service=ON\nwg_service=ON\n"
	cfgData += "vm_redir=ON\nvm_ipv4='172.16.100.0/24 172.16.101.0/24'\n"
	cfgData += "quic_rj=ON\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	gwData := "fw_wan_ports=80,443\nvms_port=8080\nsss_port=9000\nwg_ipv4=10.20.0.0/16\nwg_ipv6=fd12:3456::/48\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "gateway.cfg"), []byte(gwData), 0o644); err != nil {
		t.Fatal(err)
	}
	envData := "BINDIR=/opt/shellcrash-bin\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte(envData), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CommonPorts != "OFF" || cfg.CNIPRoute != "OFF" {
		t.Fatalf("unexpected on/off parsing: %+v", cfg)
	}
	if cfg.FwMark != 4444 || cfg.Table != 200 {
		t.Fatalf("unexpected numeric parsing: %+v", cfg)
	}
	if cfg.FWWanPorts != "80,443" || cfg.VMSPort != "8080" || cfg.SSSPort != "9000" {
		t.Fatalf("unexpected gateway parsing: %+v", cfg)
	}
	if cfg.BindDir != "/opt/shellcrash-bin" {
		t.Fatalf("unexpected bindir parsing: %+v", cfg)
	}
	if cfg.ReplaceHostV4 != "ON" || cfg.CustHostIPv4 == "" {
		t.Fatalf("unexpected host override parsing: %+v", cfg)
	}
	if cfg.ReserveIPv4 != "1.1.1.0/24" || cfg.ReserveIPv6 != "fd00::/8" {
		t.Fatalf("unexpected reserve parsing: %+v", cfg)
	}
	if cfg.TSService != "ON" || cfg.WGService != "ON" || cfg.WGIPv4 == "" || cfg.WGIPv6 == "" {
		t.Fatalf("unexpected TS/WG parsing: %+v", cfg)
	}
	if cfg.VMRedir != "ON" || !strings.Contains(cfg.VMIPv4, "172.16.100.0/24") {
		t.Fatalf("unexpected VM redir parsing: %+v", cfg)
	}
	if cfg.QuicRJ != "ON" {
		t.Fatalf("unexpected quic_rj parsing: %+v", cfg)
	}
}

func TestBuildHostRangesFromConfig(t *testing.T) {
	cfg := Config{
		CustHostIPv4:  "192.168.55.0/24,10.10.0.0/16",
		ReplaceHostV4: "ON",
		TSService:     "ON",
		WGService:     "ON",
		WGIPv4:        "10.20.0.0/16",
		WGIPv6:        "fd12:3456::/48",
	}
	v4 := buildHostIPv4([]string{"192.168.1.0/24"}, cfg)
	v6 := buildHostIPv6([]string{"fe80::/10"}, cfg)
	joined4 := strings.Join(v4, ",")
	joined6 := strings.Join(v6, ",")
	for _, want := range []string{"192.168.55.0/24", "10.10.0.0/16", "100.64.0.0/10", "10.20.0.0/16"} {
		if !strings.Contains(joined4, want) {
			t.Fatalf("missing IPv4 range %q in %q", want, joined4)
		}
	}
	for _, want := range []string{"fe80::/10", "fd7a:115c:a1e0::/48", "fd12:3456::/48"} {
		if !strings.Contains(joined6, want) {
			t.Fatalf("missing IPv6 range %q in %q", want, joined6)
		}
	}
}

func TestBuildStopCommandsIncludesCoreCleanup(t *testing.T) {
	cfg := Config{
		CommonPorts: "ON",
		MultiPort:   "22,80,443",
		DNSMode:     "redir_host",
		CNIPRoute:   "ON",
		FwMark:      7892,
		Table:       100,
		TProxyPort:  "7893",
		MixPort:     "7890",
		DBPort:      "9999",
		FWWanPorts:  "443,8443",
		VMSPort:     "",
		SSSPort:     "8388",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}, IPv6: []string{"fe80::/10"}}
	caps := capabilities{HasIPTables: true, HasIP6Tables: true, HasIPSet: true, HasNFT: true, IPTablesWait: true, IP6TablesWait: true}
	cmds := buildStopCommands(cfg, hosts, caps)
	joined := joinCommands(cmds)

	mustContain := []string{
		"iptables -w -t nat -D PREROUTING -p tcp --dport 53 -j shellcrash_dns",
		"iptables -w -D INPUT -p tcp -m multiport --dports 443,8443,8388 -j ACCEPT",
		"ip6tables -w -D INPUT -s fe80::/10 -j ACCEPT",
		"ipset destroy cn_ip",
		"nft delete table inet shellcrash",
		"ip -6 route flush table 101",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func joinCommands(cmds []commandSpec) string {
	lines := make([]string, 0, len(cmds))
	for _, c := range cmds {
		lines = append(lines, strings.TrimSpace(c.Name+" "+strings.Join(c.Args, " ")))
	}
	return strings.Join(lines, "\n")
}
