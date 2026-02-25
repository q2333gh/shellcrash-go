package firewall

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type HostInfo struct {
	IPv4      []string
	IPv6      []string
	Reserved4 []string
	Reserved6 []string
}

type capabilities struct {
	HasIPTables   bool
	HasIP6Tables  bool
	HasIPSet      bool
	HasNFT        bool
	IPTablesWait  bool
	IP6TablesWait bool
}

type commandSpec struct {
	Name string
	Args []string
}

func Stop(crashDir string) error {
	cfg, err := LoadConfig(crashDir)
	if err != nil {
		return err
	}
	hosts := detectHosts(cfg)
	caps := detectCapabilities()
	for _, c := range buildStopCommands(cfg, hosts, caps) {
		runIgnore(c.Name, c.Args...)
	}

	firewallBak := "/etc/init.d/firewall.bak"
	if st, err := os.Stat(firewallBak); err == nil && st.Size() > 0 {
		runIgnore("mv", "-f", firewallBak, "/etc/init.d/firewall")
	}
	removeResolvRepairLine("/etc/resolv.conf")
	return nil
}

func buildStopCommands(cfg Config, hosts HostInfo, caps capabilities) []commandSpec {
	cmds := make([]commandSpec, 0, 256)
	ports := make([]string, 0, 4)
	if strings.EqualFold(cfg.CommonPorts, "ON") {
		ports = []string{"-m", "multiport", "--dports", cfg.MultiPort}
	}
	acceptPorts := compactPortList(cfg.FWWanPorts, cfg.VMSPort, cfg.SSSPort)
	mixDBPorts := compactPortList(cfg.MixPort, cfg.DBPort)
	setCN := cfg.DNSMode != "fake-ip" && !strings.EqualFold(cfg.CNIPRoute, "OFF")

	if caps.HasIPTables {
		prefix := []string{}
		if caps.IPTablesWait {
			prefix = append(prefix, "-w")
		}
		add4 := func(args ...string) {
			cmds = append(cmds, commandSpec{Name: "iptables", Args: append(append([]string{}, prefix...), args...)})
		}

		add4("-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", "53", "-j", "shellcrash_dns")
		add4("-t", "nat", "-D", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "shellcrash_dns")
		add4("-t", "nat", "-D", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "shellcrash_dns_out")
		add4("-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "53", "-j", "shellcrash_dns_out")
		add4(append([]string{"-t", "nat", "-D", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrash")...)...)
		add4("-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash")
		add4(append([]string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp"}, append(ports, "-j", "shellcrash_out")...)...)
		add4("-t", "nat", "-D", "OUTPUT", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash_out")
		add4("-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", "53", "-j", "shellcrash_vm_dns")
		add4("-t", "nat", "-D", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "shellcrash_vm_dns")
		add4(append([]string{"-t", "nat", "-D", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrash_vm")...)...)
		add4("-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash_vm")

		add4(append([]string{"-t", "mangle", "-D", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrash_mark")...)...)
		add4(append([]string{"-t", "mangle", "-D", "PREROUTING", "-p", "udp"}, append(ports, "-j", "shellcrash_mark")...)...)
		add4("-t", "mangle", "-D", "PREROUTING", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash_mark")
		add4("-t", "mangle", "-D", "PREROUTING", "-p", "udp", "-d", "28.0.0.0/8", "-j", "shellcrash_mark")
		add4(append([]string{"-t", "mangle", "-D", "OUTPUT", "-p", "tcp"}, append(ports, "-j", "shellcrash_mark_out")...)...)
		add4(append([]string{"-t", "mangle", "-D", "OUTPUT", "-p", "udp"}, append(ports, "-j", "shellcrash_mark_out")...)...)
		add4("-t", "mangle", "-D", "OUTPUT", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash_mark_out")
		add4("-t", "mangle", "-D", "OUTPUT", "-p", "udp", "-d", "28.0.0.0/8", "-j", "shellcrash_mark_out")
		fwmark := strconv.Itoa(cfg.FwMark)
		add4("-t", "mangle", "-D", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "tcp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
		add4("-t", "mangle", "-D", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "udp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
		add4("-D", "FORWARD", "-o", "utun", "-j", "ACCEPT")

		if setCN {
			add4("-D", "INPUT", "-p", "udp", "--dport", "443", "-m", "set", "!", "--match-set", "cn_ip", "dst", "-j", "REJECT")
			add4("-D", "FORWARD", "-p", "udp", "--dport", "443", "-o", "utun", "-m", "set", "!", "--match-set", "cn_ip", "dst", "-j", "REJECT")
		} else {
			add4("-D", "INPUT", "-p", "udp", "--dport", "443", "-j", "REJECT")
			add4("-D", "FORWARD", "-p", "udp", "--dport", "443", "-o", "utun", "-j", "REJECT")
		}

		add4("-D", "INPUT", "-i", "lo", "-j", "ACCEPT")
		for _, ip := range hosts.IPv4 {
			add4("-D", "INPUT", "-s", ip, "-j", "ACCEPT")
		}
		if acceptPorts != "" {
			add4("-D", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
			add4("-D", "INPUT", "-p", "udp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
		}
		if mixDBPorts != "" {
			add4("-D", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", mixDBPorts, "-j", "REJECT")
			add4("-D", "INPUT", "-p", "udp", "-m", "multiport", "--dports", mixDBPorts, "-j", "REJECT")
		}
		for _, ch := range []string{"shellcrash_dns", "shellcrash", "shellcrash_out", "shellcrash_dns_out", "shellcrash_vm", "shellcrash_vm_dns"} {
			add4("-t", "nat", "-F", ch)
			add4("-t", "nat", "-X", ch)
		}
		for _, ch := range []string{"shellcrash_mark", "shellcrash_mark_out"} {
			add4("-t", "mangle", "-F", ch)
			add4("-t", "mangle", "-X", ch)
		}
	}

	if caps.HasIP6Tables {
		prefix := []string{}
		if caps.IP6TablesWait {
			prefix = append(prefix, "-w")
		}
		add6 := func(args ...string) {
			cmds = append(cmds, commandSpec{Name: "ip6tables", Args: append(append([]string{}, prefix...), args...)})
		}

		add6("-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", "53", "-j", "shellcrashv6_dns")
		add6("-t", "nat", "-D", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "shellcrashv6_dns")
		add6(append([]string{"-t", "nat", "-D", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrashv6")...)...)
		add6("-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-d", "fc00::/16", "-j", "shellcrashv6")
		add6(append([]string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp"}, append(ports, "-j", "shellcrashv6_out")...)...)
		add6("-t", "nat", "-D", "OUTPUT", "-p", "tcp", "-d", "fc00::/16", "-j", "shellcrashv6_out")
		add6("-D", "INPUT", "-p", "tcp", "--dport", "53", "-j", "REJECT")
		add6("-D", "INPUT", "-p", "udp", "--dport", "53", "-j", "REJECT")

		add6(append([]string{"-t", "mangle", "-D", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrashv6_mark")...)...)
		add6(append([]string{"-t", "mangle", "-D", "PREROUTING", "-p", "udp"}, append(ports, "-j", "shellcrashv6_mark")...)...)
		add6("-t", "mangle", "-D", "PREROUTING", "-p", "tcp", "-d", "fc00::/16", "-j", "shellcrashv6_mark")
		add6("-t", "mangle", "-D", "PREROUTING", "-p", "udp", "-d", "fc00::/16", "-j", "shellcrashv6_mark")
		add6(append([]string{"-t", "mangle", "-D", "OUTPUT", "-p", "tcp"}, append(ports, "-j", "shellcrashv6_mark_out")...)...)
		add6(append([]string{"-t", "mangle", "-D", "OUTPUT", "-p", "udp"}, append(ports, "-j", "shellcrashv6_mark_out")...)...)
		add6("-t", "mangle", "-D", "OUTPUT", "-p", "tcp", "-d", "fc00::/16", "-j", "shellcrashv6_mark_out")
		add6("-t", "mangle", "-D", "OUTPUT", "-p", "udp", "-d", "fc00::/16", "-j", "shellcrashv6_mark_out")
		fwmark := strconv.Itoa(cfg.FwMark)
		add6("-t", "mangle", "-D", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "tcp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
		add6("-t", "mangle", "-D", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "udp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
		add6("-D", "FORWARD", "-o", "utun", "-j", "ACCEPT")

		if setCN {
			add6("-D", "INPUT", "-p", "udp", "--dport", "443", "-m", "set", "!", "--match-set", "cn_ip6", "dst", "-j", "REJECT")
			add6("-D", "FORWARD", "-p", "udp", "--dport", "443", "-o", "utun", "-m", "set", "!", "--match-set", "cn_ip6", "dst", "-j", "REJECT")
		} else {
			add6("-D", "INPUT", "-p", "udp", "--dport", "443", "-j", "REJECT")
			add6("-D", "FORWARD", "-p", "udp", "--dport", "443", "-o", "utun", "-j", "REJECT")
		}

		add6("-D", "INPUT", "-i", "lo", "-j", "ACCEPT")
		for _, ip := range hosts.IPv6 {
			add6("-D", "INPUT", "-s", ip, "-j", "ACCEPT")
		}
		if acceptPorts != "" {
			add6("-D", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
			add6("-D", "INPUT", "-p", "udp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
		}
		if mixDBPorts != "" {
			add6("-D", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", mixDBPorts, "-j", "REJECT")
			add6("-D", "INPUT", "-p", "udp", "-m", "multiport", "--dports", mixDBPorts, "-j", "REJECT")
		}
		for _, ch := range []string{"shellcrashv6_dns", "shellcrashv6", "shellcrashv6_out"} {
			add6("-t", "nat", "-F", ch)
			add6("-t", "nat", "-X", ch)
		}
		for _, ch := range []string{"shellcrashv6_mark", "shellcrashv6_mark_out"} {
			add6("-t", "mangle", "-F", ch)
			add6("-t", "mangle", "-X", ch)
		}
		add6("-t", "mangle", "-F", "shellcrashv6_mark")
		add6("-t", "mangle", "-X", "shellcrashv6_mark")
	}

	if caps.HasIPSet {
		cmds = append(cmds, commandSpec{Name: "ipset", Args: []string{"destroy", "cn_ip"}})
		cmds = append(cmds, commandSpec{Name: "ipset", Args: []string{"destroy", "cn_ip6"}})
	}
	fwmark := strconv.Itoa(cfg.FwMark)
	table := strconv.Itoa(cfg.Table)
	table6 := strconv.Itoa(cfg.Table + 1)
	cmds = append(cmds,
		commandSpec{Name: "ip", Args: []string{"rule", "del", "fwmark", fwmark, "table", table}},
		commandSpec{Name: "ip", Args: []string{"route", "flush", "table", table}},
		commandSpec{Name: "ip", Args: []string{"-6", "rule", "del", "fwmark", fwmark, "table", table6}},
		commandSpec{Name: "ip", Args: []string{"-6", "route", "flush", "table", table6}},
	)
	if caps.HasNFT {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"delete", "table", "inet", "shellcrash"}})
	}
	return cmds
}

func detectCapabilities() capabilities {
	caps := capabilities{
		HasIPTables:  commandExists("iptables"),
		HasIP6Tables: commandExists("ip6tables"),
		HasIPSet:     commandExists("ipset"),
		HasNFT:       commandExists("nft"),
	}
	if caps.HasIPTables {
		caps.IPTablesWait = bytes.Contains(outputIgnore("iptables", "-h"), []byte("-w"))
	}
	if caps.HasIP6Tables {
		caps.IP6TablesWait = bytes.Contains(outputIgnore("ip6tables", "-h"), []byte("-w"))
	}
	return caps
}

func detectHosts(cfg Config) HostInfo {
	v4 := parseHostCIDRs(outputIgnore("ip", "route", "show", "scope", "link"), false)
	if len(v4) == 0 {
		v4 = []string{"192.168.0.0/16", "10.0.0.0/12", "172.16.0.0/12"}
	}
	v4 = buildHostIPv4(v4, cfg)

	v6 := []string{"fe80::/10", "fd00::/8"}
	v6 = append(v6, parseHostCIDRs(outputIgnore("ip", "-6", "route", "show", "default"), true)...)
	v6 = buildHostIPv6(v6, cfg)

	reserved4 := defaultReservedIPv4()
	if custom := parseCIDRList(cfg.ReserveIPv4); len(custom) > 0 {
		reserved4 = custom
	}
	reserved6 := defaultReservedIPv6()
	if custom := parseCIDRList(cfg.ReserveIPv6); len(custom) > 0 {
		reserved6 = custom
	}
	reserved6 = append(reserved6, v6...)
	reserved6 = dedupeNonEmpty(reserved6)

	return HostInfo{IPv4: v4, IPv6: v6, Reserved4: reserved4, Reserved6: reserved6}
}

func buildHostIPv4(base []string, cfg Config) []string {
	out := append([]string{}, base...)
	customV4 := parseCIDRList(cfg.CustHostIPv4)
	if strings.EqualFold(cfg.ReplaceHostV4, "ON") {
		if len(customV4) > 0 {
			out = customV4
		}
	} else {
		out = append(out, customV4...)
	}
	if strings.EqualFold(cfg.TSService, "ON") {
		out = append(out, "100.64.0.0/10")
	}
	if strings.EqualFold(cfg.WGService, "ON") {
		out = append(out, parseCIDRList(cfg.WGIPv4)...)
	}
	return dedupeNonEmpty(out)
}

func buildHostIPv6(base []string, cfg Config) []string {
	out := append([]string{}, base...)
	if strings.EqualFold(cfg.TSService, "ON") {
		out = append(out, "fd7a:115c:a1e0::/48")
	}
	if strings.EqualFold(cfg.WGService, "ON") {
		out = append(out, parseCIDRList(cfg.WGIPv6)...)
	}
	return dedupeNonEmpty(out)
}

func parseHostCIDRs(out []byte, ipv6 bool) []string {
	lines := strings.Split(string(out), "\n")
	items := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if !ipv6 && strings.ContainsAny(line, "\t ") {
			ifacesBlacklist := []string{"wan", "utun", "iot", "peer", "docker", "podman", "virbr", "vnet", "ovs", "vmbr", "veth", "vmnic", "vboxnet", "lxcbr", "xenbr", "vEthernet"}
			skip := false
			for _, k := range ifacesBlacklist {
				if strings.Contains(line, k) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if !ipv6 {
			if strings.Contains(fields[0], "/") {
				items = append(items, fields[0])
			}
			continue
		}
		if len(fields) >= 3 {
			items = append(items, fields[2])
		}
	}
	return dedupeNonEmpty(items)
}

func parseCIDRList(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", " ")
	return dedupeNonEmpty(strings.Fields(raw))
}

func dedupeNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func compactPortList(parts ...string) string {
	vals := make([]string, 0, len(parts))
	for _, p := range parts {
		for _, seg := range strings.Split(p, ",") {
			seg = strings.TrimSpace(seg)
			if seg != "" {
				vals = append(vals, seg)
			}
		}
	}
	if len(vals) == 0 {
		return ""
	}
	return strings.Join(dedupeNonEmpty(vals), ",")
}

func removeResolvRepairLine(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(b), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "shellcrash-dns-repair") {
			continue
		}
		filtered = append(filtered, line)
	}
	_ = os.WriteFile(path, []byte(strings.Join(filtered, "\n")), 0o644)
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runIgnore(name string, args ...string) {
	cmd := exec.Command(name, args...)
	_ = cmd.Run()
}

func outputIgnore(name string, args ...string) []byte {
	b, err := exec.Command(name, args...).Output()
	if err != nil {
		return nil
	}
	return b
}

func FirewallBakPath(crashDir string) string {
	return filepath.Join(crashDir, "..", "..", "etc", "init.d", "firewall.bak")
}
