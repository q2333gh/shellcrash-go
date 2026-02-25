package firewall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildStartCommandsAddsRoutingAndCoreRules(t *testing.T) {
	cfg := Config{
		CommonPorts:  "ON",
		MultiPort:    "22,80,443",
		DNSMode:      "mix",
		CNIPRoute:    "ON",
		FwMark:       7892,
		Table:        100,
		RedirPort:    "7892",
		DNSRedirPort: "1053",
		MixPort:      "7890",
		DBPort:       "9999",
		FWWanPorts:   "443,8443",
		SSSPort:      "8388",
		FirewallArea: "5",
		FirewallMod:  "iptables",
		RedirMod:     "Mix",
		BypassHost:   "192.168.1.1",
		FWWan:        "ON",
	}
	hosts := HostInfo{
		IPv4: []string{"192.168.1.0/24"},
		IPv6: []string{"fe80::/10"},
	}
	caps := capabilities{
		HasIPTables:   true,
		HasIP6Tables:  true,
		IPTablesWait:  true,
		IP6TablesWait: true,
	}
	cmds := buildStartCommands(cfg, hosts, caps, true)
	joined := joinCommands(cmds)

	mustContain := []string{
		"ip route add default dev utun table 100",
		"ip route add default via 192.168.1.1 table 100",
		"ip rule add fwmark 7892 table 100",
		"iptables -w -I INPUT -p tcp -m multiport --dports 443,8443,8388 -j ACCEPT",
		"iptables -w -t nat -N shellcrash",
		"iptables -w -t nat -I PREROUTING -p tcp -m multiport --dports 22,80,443 -j shellcrash",
		"iptables -w -I FORWARD -o utun -j ACCEPT",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsNoFirewallRulesWhenNotIPTables(t *testing.T) {
	cfg := Config{
		FirewallArea: "3",
		FirewallMod:  "nftables",
		RedirMod:     "Redir",
		FWWan:        "ON",
		DNSRedirPort: "1053",
		RedirPort:    "7892",
		Table:        100,
		FwMark:       7892,
		MixPort:      "7890",
		DBPort:       "9999",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}, IPv6: []string{"fe80::/10"}}
	caps := capabilities{HasIPTables: true, HasNFT: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)
	if strings.Contains(joined, "iptables ") {
		t.Fatalf("expected no iptables rules for nftables mode, got:\n%s", joined)
	}
	mustContain := []string{
		"nft add table inet shellcrash",
		"nft flush table inet shellcrash",
		"nft add chain inet shellcrash prerouting { type nat hook prerouting priority -100 ; }",
		"nft add chain inet shellcrash prerouting_dns { type nat hook prerouting priority -100 ; }",
		"nft add rule inet shellcrash prerouting meta l4proto tcp redirect to 7892",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsDNSChainsForAreaThree(t *testing.T) {
	cfg := Config{
		FirewallArea: "3",
		FirewallMod:  "iptables",
		RedirMod:     "Redir",
		FWWan:        "OFF",
		DNSRedirPort: "1053",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}}
	caps := capabilities{HasIPTables: true, IPTablesWait: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)
	mustContain := []string{
		"iptables -w -t nat -N shellcrash_dns",
		"iptables -w -t nat -I PREROUTING -p tcp --dport 53 -j shellcrash_dns",
		"iptables -w -t nat -N shellcrash_dns_out",
		"iptables -w -t nat -I OUTPUT -p udp --dport 53 -j shellcrash_dns_out",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsTProxyMangleRules(t *testing.T) {
	cfg := Config{
		CommonPorts:  "ON",
		MultiPort:    "80,443",
		DNSMode:      "mix",
		CNIPRoute:    "ON",
		FwMark:       7892,
		Table:        100,
		TProxyPort:   "7893",
		MixPort:      "7890",
		RedirPort:    "7892",
		FirewallArea: "3",
		FirewallMod:  "iptables",
		RedirMod:     "Tproxy",
		IPv6Redir:    "ON",
		CrashCore:    "meta",
		FWWan:        "OFF",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}, IPv6: []string{"fe80::/10"}}
	caps := capabilities{HasIPTables: true, HasIP6Tables: true, IPTablesWait: true, IP6TablesWait: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)

	mustContain := []string{
		"iptables -w -t mangle -N shellcrash_mark",
		"iptables -w -t mangle -A shellcrash_mark -m mark --mark 7894 -j RETURN",
		"iptables -w -t mangle -A shellcrash_mark -p tcp -j TPROXY --on-port 7893 --tproxy-mark 7892",
		"iptables -w -t mangle -N shellcrash_mark_out",
		"iptables -w -t mangle -A shellcrash_mark_out -m owner --gid-owner 7890 -j RETURN",
		"iptables -w -t mangle -A shellcrash_mark_out -p tcp -j MARK --set-mark 7892",
		"iptables -w -t mangle -A PREROUTING -m mark --mark 7892 -p udp -j TPROXY --on-port 7893",
		"ip6tables -w -t mangle -N shellcrashv6_mark",
		"ip6tables -w -t mangle -N shellcrashv6_mark_out",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsNFTCNSetsAndBlacklistFilters(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(td, "bin")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "mac"), []byte("aa:bb:cc:dd:ee:ff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ip_filter"), []byte("192.168.1.50/32\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "cn_ip.txt"), []byte("1.1.1.0/24\n2.2.2.0/24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "cn_ipv6.txt"), []byte("2400:3200::/32\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		CrashDir:      crashDir,
		BindDir:       binDir,
		FirewallArea:  "1",
		FirewallMod:   "nftables",
		RedirMod:      "Redir",
		FWWan:         "OFF",
		DNSMode:       "redir_host",
		CNIPRoute:     "ON",
		MacFilterType: "黑名单",
		RedirPort:     "7892",
		DNSRedirPort:  "1053",
		FwMark:        7892,
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}, IPv6: []string{"fe80::/10"}}
	caps := capabilities{HasNFT: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)

	mustContain := []string{
		"nft add set inet shellcrash cn_ip { type ipv4_addr ; flags interval ; }",
		"nft add element inet shellcrash cn_ip { 1.1.1.0/24, 2.2.2.0/24 }",
		"nft add set inet shellcrash cn_ip6 { type ipv6_addr ; flags interval ; }",
		"nft add element inet shellcrash cn_ip6 { 2400:3200::/32 }",
		"nft add rule inet shellcrash prerouting ether saddr { aa:bb:cc:dd:ee:ff } return",
		"nft add rule inet shellcrash prerouting ip saddr { 192.168.1.50/32 } return",
		"nft add rule inet shellcrash prerouting ip saddr != { 192.168.1.0/24 } return",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsNFTWhitelistFilter(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "mac"), []byte("aa:bb:cc:dd:ee:ff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ip_filter"), []byte("192.168.1.50/32\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		CrashDir:      crashDir,
		BindDir:       filepath.Join(td, "bin"),
		FirewallArea:  "1",
		FirewallMod:   "nftables",
		RedirMod:      "Redir",
		FWWan:         "OFF",
		DNSMode:       "redir_host",
		CNIPRoute:     "OFF",
		MacFilterType: "白名单",
		RedirPort:     "7892",
		DNSRedirPort:  "1053",
		FwMark:        7892,
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}}
	caps := capabilities{HasNFT: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)
	needle := "nft add rule inet shellcrash prerouting ether saddr != { aa:bb:cc:dd:ee:ff } ip saddr != { 192.168.1.50/32 } return"
	if !strings.Contains(joined, needle) {
		t.Fatalf("expected whitelist filter rule %q in plan\n%s", needle, joined)
	}
}

func TestBuildStartCommandsIPTablesVMRedir(t *testing.T) {
	cfg := Config{
		CommonPorts:  "ON",
		MultiPort:    "80,443",
		DNSMode:      "mix",
		FwMark:       7892,
		FirewallArea: "1",
		FirewallMod:  "iptables",
		RedirMod:     "Redir",
		RedirPort:    "7892",
		DNSRedirPort: "1053",
		VMRedir:      "ON",
		VMIPv4:       "172.16.100.0/24 172.16.101.0/24",
		FWWan:        "OFF",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}}
	caps := capabilities{HasIPTables: true, IPTablesWait: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)

	mustContain := []string{
		"iptables -w -t nat -N shellcrash_vm_dns",
		"iptables -w -t nat -A shellcrash_vm_dns -s 172.16.100.0/24 -p tcp --dport 53 -j REDIRECT --to-ports 1053",
		"iptables -w -t nat -N shellcrash_vm",
		"iptables -w -t nat -A shellcrash_vm -m mark --mark 7894 -j RETURN",
		"iptables -w -t nat -A shellcrash_vm -s 172.16.101.0/24 -p tcp -j REDIRECT --to-ports 7892",
		"iptables -w -t nat -I PREROUTING -p tcp -m multiport --dports 80,443 -j shellcrash_vm",
		"iptables -w -t nat -I PREROUTING -p tcp -d 28.0.0.0/8 -j shellcrash_vm",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsNFTVMRedir(t *testing.T) {
	cfg := Config{
		DNSMode:      "redir_host",
		FwMark:       7892,
		FirewallArea: "1",
		FirewallMod:  "nftables",
		RedirMod:     "Redir",
		RedirPort:    "7892",
		DNSRedirPort: "1053",
		VMRedir:      "ON",
		VMIPv4:       "172.16.100.0/24 172.16.101.0/24",
		FWWan:        "OFF",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}}
	caps := capabilities{HasNFT: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)

	mustContain := []string{
		"nft add chain inet shellcrash prerouting_vm_dns { type nat hook prerouting priority -100 ; }",
		"nft add rule inet shellcrash prerouting_vm_dns ip saddr != { 172.16.100.0/24, 172.16.101.0/24 } return",
		"nft add chain inet shellcrash prerouting_vm { type nat hook prerouting priority -100 ; }",
		"nft add rule inet shellcrash prerouting_vm ip saddr != { 172.16.100.0/24, 172.16.101.0/24 } return",
		"nft add rule inet shellcrash prerouting_vm meta l4proto tcp redirect to 7892",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsIPTablesQUICReject(t *testing.T) {
	cfg := Config{
		DNSMode:      "redir_host",
		CNIPRoute:    "ON",
		FirewallArea: "1",
		FirewallMod:  "iptables",
		RedirMod:     "Tun",
		QuicRJ:       "ON",
		FWWan:        "OFF",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}, IPv6: []string{"fe80::/10"}}
	caps := capabilities{HasIPTables: true, HasIP6Tables: true, IPTablesWait: true, IP6TablesWait: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)

	mustContain := []string{
		"iptables -w -I FORWARD -p udp --dport 443 -o utun -m set ! --match-set cn_ip dst -j REJECT",
		"ip6tables -w -I FORWARD -p udp --dport 443 -o utun -m set ! --match-set cn_ip6 dst -j REJECT",
	}
	for _, needle := range mustContain {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected command %q in plan\n%s", needle, joined)
		}
	}
}

func TestBuildStartCommandsNFTQUICBypassRule(t *testing.T) {
	cfg := Config{
		DNSMode:      "redir_host",
		FirewallArea: "1",
		FirewallMod:  "nftables",
		RedirMod:     "Tproxy",
		QuicRJ:       "ON",
		RedirPort:    "7892",
		TProxyPort:   "7893",
		FwMark:       7892,
		FWWan:        "OFF",
	}
	hosts := HostInfo{IPv4: []string{"192.168.1.0/24"}}
	caps := capabilities{HasNFT: true}
	cmds := buildStartCommands(cfg, hosts, caps, false)
	joined := joinCommands(cmds)
	needle := "nft add rule inet shellcrash prerouting udp dport { 443, 8443 } return"
	if !strings.Contains(joined, needle) {
		t.Fatalf("expected QUIC bypass rule %q in plan\n%s", needle, joined)
	}
}
