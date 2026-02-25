package firewall

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Start(crashDir string) error {
	cfg, err := LoadConfig(crashDir)
	if err != nil {
		return err
	}
	hosts := detectHosts(cfg)
	caps := detectCapabilities()
	for _, c := range buildStartCommands(cfg, hosts, caps, hasUTunRoute()) {
		runIgnore(c.Name, c.Args...)
	}
	ensureResolvRepairLine(cfg, "/etc/resolv.conf")
	clearOpenWrtDNSRedirect()
	return nil
}

func buildStartCommands(cfg Config, hosts HostInfo, caps capabilities, hasUTun bool) []commandSpec {
	cmds := make([]commandSpec, 0, 256)
	if len(hosts.Reserved4) == 0 {
		hosts.Reserved4 = defaultReservedIPv4()
	}
	if len(hosts.Reserved6) == 0 {
		hosts.Reserved6 = append(defaultReservedIPv6(), hosts.IPv6...)
		hosts.Reserved6 = dedupeNonEmpty(hosts.Reserved6)
	}
	area := atoiDefault(cfg.FirewallArea, 1)
	lanProxy := area == 1 || area == 3 || area == 5
	localProxy := area == 2 || area == 3
	setCN := cfg.DNSMode != "fake-ip" && !strings.EqualFold(cfg.CNIPRoute, "OFF")
	rejectPorts := compactPortList(cfg.MixPort, cfg.DBPort)
	acceptPorts := compactPortList(cfg.FWWanPorts, cfg.VMSPort, cfg.SSSPort)

	if area != 4 {
		if cfg.RedirMod == "Tproxy" {
			cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"route", "add", "local", "default", "dev", "lo", "table", strconv.Itoa(cfg.Table)}})
		}
		if cfg.RedirMod == "Tun" || cfg.RedirMod == "Mix" {
			if hasUTun {
				cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"route", "add", "default", "dev", "utun", "table", strconv.Itoa(cfg.Table)}})
			}
		}
		if area == 5 && cfg.BypassHost != "" {
			cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"route", "add", "default", "via", cfg.BypassHost, "table", strconv.Itoa(cfg.Table)}})
		}
		if cfg.RedirMod != "Redir" {
			cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"rule", "add", "fwmark", strconv.Itoa(cfg.FwMark), "table", strconv.Itoa(cfg.Table)}})
		}
	}

	if strings.EqualFold(cfg.IPv6Redir, "ON") && area <= 3 {
		table6 := strconv.Itoa(cfg.Table + 1)
		if cfg.RedirMod == "Tproxy" {
			cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"-6", "route", "add", "local", "default", "dev", "lo", "table", table6}})
		}
		if hasUTun {
			cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"-6", "route", "add", "default", "dev", "utun", "table", table6}})
		}
		if cfg.RedirMod != "Redir" {
			cmds = append(cmds, commandSpec{Name: "ip", Args: []string{"-6", "rule", "add", "fwmark", strconv.Itoa(cfg.FwMark), "table", table6}})
		}
	}

	if !strings.EqualFold(cfg.FWWan, "OFF") && !strings.EqualFold(cfg.FirewallMod, "nftables") {
		cmds = appendWANRules(cmds, cfg, hosts, caps, acceptPorts, rejectPorts)
	}

	if strings.EqualFold(cfg.FirewallMod, "nftables") && caps.HasNFT {
		cmds = appendNFTRules(cmds, cfg, hosts, lanProxy, localProxy, setCN)
		cmds = appendQUICRejectRules(cmds, cfg, caps, lanProxy, setCN)
		cmds = appendVMRedirRules(cmds, cfg, caps)
		if cfg.RedirMod == "Tun" || cfg.RedirMod == "Mix" {
			cmds = appendNFTTunForwardAccept(cmds)
		}
		return cmds
	}

	if cfg.FirewallMod != "iptables" || !caps.HasIPTables {
		return cmds
	}
	if cfg.RedirMod == "Redir" || cfg.RedirMod == "Mix" {
		cmds = appendIPTablesRedirRules(cmds, cfg, hosts, caps, lanProxy, localProxy, setCN)
	}
	if cfg.RedirMod == "Tproxy" || cfg.RedirMod == "Tun" || cfg.RedirMod == "Mix" || cfg.RedirMod == "T&U旁路转发" || cfg.RedirMod == "TCP旁路转发" {
		cmds = appendIPTablesMangleRules(cmds, cfg, hosts, caps, lanProxy, localProxy, setCN)
	}
	if area <= 3 {
		cmds = appendIPTablesDNSRules(cmds, cfg, caps, lanProxy, localProxy)
	}
	cmds = appendQUICRejectRules(cmds, cfg, caps, lanProxy, setCN)
	if cfg.RedirMod == "Tun" || cfg.RedirMod == "Mix" {
		cmds = appendTunForwardAccept(cmds, caps)
	}
	cmds = appendVMRedirRules(cmds, cfg, caps)
	return cmds
}

func appendQUICRejectRules(cmds []commandSpec, cfg Config, caps capabilities, lanProxy bool, setCN bool) []commandSpec {
	if !strings.EqualFold(cfg.QuicRJ, "ON") || !lanProxy {
		return cmds
	}
	if strings.EqualFold(cfg.FirewallMod, "nftables") && caps.HasNFT {
		return cmds
	}
	if !caps.HasIPTables || cfg.RedirMod == "Redir" {
		return cmds
	}

	add4 := func(args ...string) {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, args...)})
	}
	add6 := func(args ...string) {
		if !caps.HasIP6Tables {
			return
		}
		base := []string{}
		if caps.IP6TablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "ip6tables", Args: append(base, args...)})
	}

	appendSetCN := func(args []string, ipv6 bool) []string {
		if !setCN {
			return args
		}
		if ipv6 {
			return append(args, "-m", "set", "!", "--match-set", "cn_ip6", "dst")
		}
		return append(args, "-m", "set", "!", "--match-set", "cn_ip", "dst")
	}

	switch cfg.RedirMod {
	case "Tun", "Mix":
		add4(append(appendSetCN([]string{"-I", "FORWARD", "-p", "udp", "--dport", "443", "-o", "utun"}, false), "-j", "REJECT")...)
		add6(append(appendSetCN([]string{"-I", "FORWARD", "-p", "udp", "--dport", "443", "-o", "utun"}, true), "-j", "REJECT")...)
	case "Tproxy":
		add4(append(appendSetCN([]string{"-I", "INPUT", "-p", "udp", "--dport", "443"}, false), "-j", "REJECT")...)
		add6(append(appendSetCN([]string{"-I", "INPUT", "-p", "udp", "--dport", "443"}, true), "-j", "REJECT")...)
	}
	return cmds
}

func appendIPTablesMangleRules(cmds []commandSpec, cfg Config, hosts HostInfo, caps capabilities, lanProxy bool, localProxy bool, setCN bool) []commandSpec {
	ports := []string{}
	if strings.EqualFold(cfg.CommonPorts, "ON") {
		ports = []string{"-m", "multiport", "--dports", cfg.MultiPort}
	}
	routingMark := strconv.Itoa(cfg.FwMark + 2)
	fwmark := strconv.Itoa(cfg.FwMark)
	redirPorts := compactPortList(cfg.MixPort, cfg.RedirPort, cfg.TProxyPort)

	add4 := func(args ...string) {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, args...)})
	}
	add6 := func(args ...string) {
		if !caps.HasIP6Tables {
			return
		}
		base := []string{}
		if caps.IP6TablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "ip6tables", Args: append(base, args...)})
	}

	type markMode struct {
		proto       []string
		localTPROXY bool
	}
	mark := markMode{}
	switch cfg.RedirMod {
	case "Tproxy":
		mark.proto = []string{"tcp", "udp"}
		mark.localTPROXY = true
	case "Tun", "T&U旁路转发":
		mark.proto = []string{"tcp", "udp"}
	case "Mix":
		mark.proto = []string{"udp"}
	case "TCP旁路转发":
		mark.proto = []string{"tcp"}
	}

	buildV4Chain := func(chain, srcTableChain string, hostCIDRs []string, localOutput bool) {
		add4("-t", "mangle", "-N", chain)
		add4("-t", "mangle", "-A", chain, "-p", "tcp", "--dport", "53", "-j", "RETURN")
		add4("-t", "mangle", "-A", chain, "-p", "udp", "--dport", "53", "-j", "RETURN")
		add4("-t", "mangle", "-A", chain, "-m", "mark", "--mark", routingMark, "-j", "RETURN")
		if localOutput {
			add4("-t", "mangle", "-A", chain, "-m", "owner", "--gid-owner", "453", "-j", "RETURN")
			add4("-t", "mangle", "-A", chain, "-m", "owner", "--gid-owner", "7890", "-j", "RETURN")
		}
		if redirPorts != "" && len(ports) == 0 {
			add4("-t", "mangle", "-A", chain, "-p", "tcp", "-m", "multiport", "--dports", redirPorts, "-j", "RETURN")
			add4("-t", "mangle", "-A", chain, "-p", "udp", "-m", "multiport", "--dports", redirPorts, "-j", "RETURN")
		}
		for _, dst := range append(append([]string{}, hostCIDRs...), hosts.Reserved4...) {
			add4("-t", "mangle", "-A", chain, "-d", dst, "-j", "RETURN")
		}
		if setCN {
			add4("-t", "mangle", "-A", chain, "-m", "set", "--match-set", "cn_ip", "dst", "-j", "RETURN")
		}

		for _, proto := range mark.proto {
			if cfg.RedirMod == "Tproxy" && !localOutput {
				add4("-t", "mangle", "-A", chain, "-p", proto, "-j", "TPROXY", "--on-port", cfg.TProxyPort, "--tproxy-mark", fwmark)
			} else {
				add4("-t", "mangle", "-A", chain, "-p", proto, "-j", "MARK", "--set-mark", fwmark)
			}
			add4(append([]string{"-t", "mangle", "-I", srcTableChain, "-p", proto}, append(ports, "-j", chain)...)...)
			if strings.EqualFold(cfg.CommonPorts, "ON") && (cfg.DNSMode == "mix" || cfg.DNSMode == "fake-ip") {
				add4("-t", "mangle", "-I", srcTableChain, "-p", proto, "-d", "28.0.0.0/8", "-j", chain)
			}
		}
	}

	buildV6Chain := func(chain, srcTableChain string, hostCIDRs []string, localOutput bool) {
		add6("-t", "mangle", "-N", chain)
		add6("-t", "mangle", "-A", chain, "-p", "tcp", "--dport", "53", "-j", "RETURN")
		add6("-t", "mangle", "-A", chain, "-p", "udp", "--dport", "53", "-j", "RETURN")
		add6("-t", "mangle", "-A", chain, "-m", "mark", "--mark", routingMark, "-j", "RETURN")
		if localOutput {
			add6("-t", "mangle", "-A", chain, "-m", "owner", "--gid-owner", "453", "-j", "RETURN")
			add6("-t", "mangle", "-A", chain, "-m", "owner", "--gid-owner", "7890", "-j", "RETURN")
		}
		if redirPorts != "" && len(ports) == 0 {
			add6("-t", "mangle", "-A", chain, "-p", "tcp", "-m", "multiport", "--dports", redirPorts, "-j", "RETURN")
			add6("-t", "mangle", "-A", chain, "-p", "udp", "-m", "multiport", "--dports", redirPorts, "-j", "RETURN")
		}
		for _, dst := range append(append([]string{}, hostCIDRs...), hosts.Reserved6...) {
			add6("-t", "mangle", "-A", chain, "-d", dst, "-j", "RETURN")
		}
		if setCN {
			add6("-t", "mangle", "-A", chain, "-m", "set", "--match-set", "cn_ip6", "dst", "-j", "RETURN")
		}

		for _, proto := range mark.proto {
			if cfg.RedirMod == "Tproxy" && !localOutput {
				add6("-t", "mangle", "-A", chain, "-p", proto, "-j", "TPROXY", "--on-port", cfg.TProxyPort, "--tproxy-mark", fwmark)
			} else {
				add6("-t", "mangle", "-A", chain, "-p", proto, "-j", "MARK", "--set-mark", fwmark)
			}
			add6(append([]string{"-t", "mangle", "-I", srcTableChain, "-p", proto}, append(ports, "-j", chain)...)...)
			if strings.EqualFold(cfg.CommonPorts, "ON") && (cfg.DNSMode == "mix" || cfg.DNSMode == "fake-ip") {
				add6("-t", "mangle", "-I", srcTableChain, "-p", proto, "-d", "fc00::/16", "-j", chain)
			}
		}
	}

	if lanProxy {
		buildV4Chain("shellcrash_mark", "PREROUTING", hosts.IPv4, false)
	}
	if localProxy {
		buildV4Chain("shellcrash_mark_out", "OUTPUT", append([]string{"127.0.0.0/8"}, hosts.IPv4...), true)
		if mark.localTPROXY {
			add4("-t", "mangle", "-A", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "tcp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
			add4("-t", "mangle", "-A", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "udp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
		}
	}

	if strings.EqualFold(cfg.IPv6Redir, "ON") && !strings.EqualFold(cfg.CrashCore, "clashpre") {
		if lanProxy {
			buildV6Chain("shellcrashv6_mark", "PREROUTING", hosts.IPv6, false)
		}
		if localProxy {
			buildV6Chain("shellcrashv6_mark_out", "OUTPUT", append([]string{"::1"}, hosts.IPv6...), true)
			if mark.localTPROXY {
				add6("-t", "mangle", "-A", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "tcp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
				add6("-t", "mangle", "-A", "PREROUTING", "-m", "mark", "--mark", fwmark, "-p", "udp", "-j", "TPROXY", "--on-port", cfg.TProxyPort)
			}
		}
	}
	return cmds
}

func defaultReservedIPv4() []string {
	return []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"127.0.0.0/8",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"224.0.0.0/4",
		"240.0.0.0/4",
	}
}

func defaultReservedIPv6() []string {
	return []string{
		"::/128",
		"::1/128",
		"::ffff:0:0/96",
		"64:ff9b::/96",
		"100::/64",
		"2001::/32",
		"2001:20::/28",
		"2001:db8::/32",
		"2002::/16",
		"fe80::/10",
		"ff00::/8",
	}
}

func appendWANRules(cmds []commandSpec, cfg Config, hosts HostInfo, caps capabilities, acceptPorts string, rejectPorts string) []commandSpec {
	add4 := func(args ...string) {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, args...)})
	}
	add4("-I", "INPUT", "-i", "lo", "-j", "ACCEPT")
	for _, ip := range hosts.IPv4 {
		add4("-I", "INPUT", "-s", ip, "-j", "ACCEPT")
	}
	if acceptPorts != "" {
		add4("-I", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
		add4("-I", "INPUT", "-p", "udp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
	}
	if rejectPorts != "" {
		add4("-I", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", rejectPorts, "-j", "REJECT")
		add4("-I", "INPUT", "-p", "udp", "-m", "multiport", "--dports", rejectPorts, "-j", "REJECT")
	}

	if !caps.HasIP6Tables {
		return cmds
	}
	add6 := func(args ...string) {
		base := []string{}
		if caps.IP6TablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "ip6tables", Args: append(base, args...)})
	}
	add6("-I", "INPUT", "-i", "lo", "-j", "ACCEPT")
	for _, ip := range hosts.IPv6 {
		add6("-I", "INPUT", "-s", ip, "-j", "ACCEPT")
	}
	if acceptPorts != "" {
		add6("-I", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
		add6("-I", "INPUT", "-p", "udp", "-m", "multiport", "--dports", acceptPorts, "-j", "ACCEPT")
	}
	if rejectPorts != "" {
		add6("-I", "INPUT", "-p", "tcp", "-m", "multiport", "--dports", rejectPorts, "-j", "REJECT")
		add6("-I", "INPUT", "-p", "udp", "-m", "multiport", "--dports", rejectPorts, "-j", "REJECT")
	}
	return cmds
}

func appendIPTablesRedirRules(cmds []commandSpec, cfg Config, hosts HostInfo, caps capabilities, lanProxy bool, localProxy bool, setCN bool) []commandSpec {
	ports := []string{}
	if strings.EqualFold(cfg.CommonPorts, "ON") {
		ports = []string{"-m", "multiport", "--dports", cfg.MultiPort}
	}
	add4 := func(args ...string) {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, args...)})
	}
	makeRedirChain := func(chain string, hostCIDRs []string) {
		add4("-t", "nat", "-N", chain)
		add4("-t", "nat", "-A", chain, "-p", "tcp", "--dport", "53", "-j", "RETURN")
		add4("-t", "nat", "-A", chain, "-m", "mark", "--mark", strconv.Itoa(cfg.FwMark), "-j", "RETURN")
		for _, ip := range hostCIDRs {
			add4("-t", "nat", "-A", chain, "-s", ip, "-p", "tcp", "-j", "REDIRECT", "--to-ports", cfg.RedirPort)
		}
		if setCN {
			add4("-t", "nat", "-A", chain, "-m", "set", "--match-set", "cn_ip", "dst", "-j", "RETURN")
		}
		add4("-t", "nat", "-A", chain, "-j", "RETURN")
	}
	if lanProxy {
		makeRedirChain("shellcrash", hosts.IPv4)
		add4(append([]string{"-t", "nat", "-I", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrash")...)...)
		if cfg.DNSMode == "mix" || cfg.DNSMode == "fake-ip" {
			add4("-t", "nat", "-I", "PREROUTING", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash")
		}
	}
	if localProxy {
		makeRedirChain("shellcrash_out", append([]string{"127.0.0.0/8"}, hosts.IPv4...))
		add4(append([]string{"-t", "nat", "-I", "OUTPUT", "-p", "tcp"}, append(ports, "-j", "shellcrash_out")...)...)
		if cfg.DNSMode == "mix" || cfg.DNSMode == "fake-ip" {
			add4("-t", "nat", "-I", "OUTPUT", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash_out")
		}
	}
	return cmds
}

func appendIPTablesDNSRules(cmds []commandSpec, cfg Config, caps capabilities, lanProxy bool, localProxy bool) []commandSpec {
	add4 := func(args ...string) {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, args...)})
	}
	if lanProxy {
		add4("-t", "nat", "-N", "shellcrash_dns")
		add4("-t", "nat", "-A", "shellcrash_dns", "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-ports", cfg.DNSRedirPort)
		add4("-t", "nat", "-A", "shellcrash_dns", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", cfg.DNSRedirPort)
		add4("-t", "nat", "-I", "PREROUTING", "-p", "tcp", "--dport", "53", "-j", "shellcrash_dns")
		add4("-t", "nat", "-I", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "shellcrash_dns")
	}
	if localProxy {
		add4("-t", "nat", "-N", "shellcrash_dns_out")
		add4("-t", "nat", "-A", "shellcrash_dns_out", "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-ports", cfg.DNSRedirPort)
		add4("-t", "nat", "-A", "shellcrash_dns_out", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", cfg.DNSRedirPort)
		add4("-t", "nat", "-I", "OUTPUT", "-p", "tcp", "--dport", "53", "-j", "shellcrash_dns_out")
		add4("-t", "nat", "-I", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "shellcrash_dns_out")
	}
	return cmds
}

func appendVMRedirRules(cmds []commandSpec, cfg Config, caps capabilities) []commandSpec {
	if !strings.EqualFold(cfg.VMRedir, "ON") {
		return cmds
	}
	vmCIDRs := parseCIDRList(cfg.VMIPv4)
	if len(vmCIDRs) == 0 {
		return cmds
	}
	routingMark := strconv.Itoa(cfg.FwMark + 2)
	ports := []string{}
	if strings.EqualFold(cfg.CommonPorts, "ON") {
		ports = []string{"-m", "multiport", "--dports", cfg.MultiPort}
	}

	add4 := func(args ...string) {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, args...)})
	}

	if strings.EqualFold(cfg.FirewallMod, "iptables") && caps.HasIPTables {
		add4("-t", "nat", "-N", "shellcrash_vm_dns")
		for _, src := range vmCIDRs {
			add4("-t", "nat", "-A", "shellcrash_vm_dns", "-s", src, "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-ports", cfg.DNSRedirPort)
			add4("-t", "nat", "-A", "shellcrash_vm_dns", "-s", src, "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", cfg.DNSRedirPort)
		}
		add4("-t", "nat", "-I", "PREROUTING", "-p", "tcp", "--dport", "53", "-j", "shellcrash_vm_dns")
		add4("-t", "nat", "-I", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "shellcrash_vm_dns")

		add4("-t", "nat", "-N", "shellcrash_vm")
		add4("-t", "nat", "-A", "shellcrash_vm", "-p", "tcp", "--dport", "53", "-j", "RETURN")
		add4("-t", "nat", "-A", "shellcrash_vm", "-p", "udp", "--dport", "53", "-j", "RETURN")
		add4("-t", "nat", "-A", "shellcrash_vm", "-m", "mark", "--mark", routingMark, "-j", "RETURN")
		if len(ports) == 0 {
			redirPorts := compactPortList(cfg.MixPort, cfg.RedirPort, cfg.TProxyPort)
			if redirPorts != "" {
				add4("-t", "nat", "-A", "shellcrash_vm", "-p", "tcp", "-m", "multiport", "--dports", redirPorts, "-j", "RETURN")
				add4("-t", "nat", "-A", "shellcrash_vm", "-p", "udp", "-m", "multiport", "--dports", redirPorts, "-j", "RETURN")
			}
		}
		for _, src := range vmCIDRs {
			add4("-t", "nat", "-A", "shellcrash_vm", "-s", src, "-p", "tcp", "-j", "REDIRECT", "--to-ports", cfg.RedirPort)
		}
		if len(ports) > 0 {
			add4(append([]string{"-t", "nat", "-I", "PREROUTING", "-p", "tcp"}, append(ports, "-j", "shellcrash_vm")...)...)
			if cfg.DNSMode == "mix" || cfg.DNSMode == "fake-ip" {
				add4("-t", "nat", "-I", "PREROUTING", "-p", "tcp", "-d", "28.0.0.0/8", "-j", "shellcrash_vm")
			}
		} else {
			add4("-t", "nat", "-I", "PREROUTING", "-p", "tcp", "-j", "shellcrash_vm")
		}
	}

	if strings.EqualFold(cfg.FirewallMod, "nftables") && caps.HasNFT {
		vmSet := joinForNFTSet(vmCIDRs)
		cmds = append(cmds,
			commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", "prerouting_vm_dns", "{", "type", "nat", "hook", "prerouting", "priority", "-100", ";", "}"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "udp", "dport", "!=", "53", "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "tcp", "dport", "!=", "53", "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "meta", "mark", routingMark, "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "meta", "skgid", "{", "453,", "7890", "}", "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "ip", "saddr", "!=", "{", vmSet, "}", "return"}},
		)
		cmds = append(cmds,
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "udp", "dport", "53", "redirect", "to", cfg.DNSRedirPort}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm_dns", "tcp", "dport", "53", "redirect", "to", cfg.DNSRedirPort}},
		)

		cmds = append(cmds,
			commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", "prerouting_vm", "{", "type", "nat", "hook", "prerouting", "priority", "-100", ";", "}"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm", "tcp", "dport", "53", "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm", "udp", "dport", "53", "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm", "meta", "mark", routingMark, "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm", "meta", "skgid", "7890", "return"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm", "ip", "saddr", "!=", "{", vmSet, "}", "return"}},
		)
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_vm", "meta", "l4proto", "tcp", "redirect", "to", cfg.RedirPort}})
	}

	return cmds
}

func appendTunForwardAccept(cmds []commandSpec, caps capabilities) []commandSpec {
	if caps.HasIPTables {
		base := []string{}
		if caps.IPTablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "iptables", Args: append(base, "-I", "FORWARD", "-o", "utun", "-j", "ACCEPT")})
	}
	if caps.HasIP6Tables {
		base := []string{}
		if caps.IP6TablesWait {
			base = append(base, "-w")
		}
		cmds = append(cmds, commandSpec{Name: "ip6tables", Args: append(base, "-I", "FORWARD", "-o", "utun", "-j", "ACCEPT")})
	}
	return cmds
}

func appendNFTRules(cmds []commandSpec, cfg Config, hosts HostInfo, lanProxy bool, localProxy bool, setCN bool) []commandSpec {
	cmds = append(cmds,
		commandSpec{Name: "nft", Args: []string{"add", "table", "inet", "shellcrash"}},
		commandSpec{Name: "nft", Args: []string{"flush", "table", "inet", "shellcrash"}},
	)
	if setCN {
		cmds = appendNFTCNSets(cmds, cfg)
	}

	area := atoiDefault(cfg.FirewallArea, 1)
	if !strings.EqualFold(cfg.FWWan, "OFF") {
		rejectPorts := compactPortList(cfg.MixPort, cfg.DBPort)
		acceptPorts := compactPortList(cfg.FWWanPorts, cfg.VMSPort, cfg.SSSPort)
		cmds = append(cmds,
			commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", "input", "{", "type", "filter", "hook", "input", "priority", "-100", ";", "}"}},
			commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "iif", "lo", "accept"}},
		)
		for _, ip := range hosts.IPv4 {
			cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "ip", "saddr", ip, "accept"}})
		}
		for _, ip := range hosts.IPv6 {
			cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "ip6", "saddr", ip, "accept"}})
		}
		if acceptPorts != "" {
			cmds = append(cmds,
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "tcp", "dport", "{", strings.ReplaceAll(acceptPorts, ",", ", "), "}", "meta", "mark", "set", "0x67890", "accept"}},
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "udp", "dport", "{", strings.ReplaceAll(acceptPorts, ",", ", "), "}", "meta", "mark", "set", "0x67890", "accept"}},
			)
		}
		if rejectPorts != "" {
			cmds = append(cmds,
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "tcp", "dport", "{", strings.ReplaceAll(rejectPorts, ",", ", "), "}", "reject"}},
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "input", "udp", "dport", "{", strings.ReplaceAll(rejectPorts, ",", ", "), "}", "reject"}},
			)
		}
	}

	if area <= 3 {
		if lanProxy {
			cmds = appendNFTDNSChain(cmds, cfg, hosts, "prerouting_dns", "prerouting")
		}
		if localProxy {
			cmds = appendNFTDNSChain(cmds, cfg, hosts, "output_dns", "output")
		}
	}

	switch cfg.RedirMod {
	case "Redir":
		if lanProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "prerouting", "prerouting", "nat", "-100", "tcp", "redirect", "to", cfg.RedirPort, false, setCN)
		}
		if localProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "output", "output", "nat", "-100", "tcp", "redirect", "to", cfg.RedirPort, true, setCN)
		}
	case "Tproxy":
		if lanProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "prerouting", "prerouting", "filter", "-150", "{ tcp, udp }", "mark", "set", strconv.Itoa(cfg.FwMark), "tproxy", "to", ":"+cfg.TProxyPort, false, setCN)
		}
		if localProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "output", "output", "route", "-150", "{ tcp, udp }", "mark", "set", strconv.Itoa(cfg.FwMark), true, setCN)
			cmds = append(cmds,
				commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", "mark_out", "{", "type", "filter", "hook", "prerouting", "priority", "-100", ";", "}"}},
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "mark_out", "meta", "mark", strconv.Itoa(cfg.FwMark), "meta", "l4proto", "{", "tcp,", "udp", "}", "tproxy", "to", ":" + cfg.TProxyPort}},
			)
		}
	case "Tun":
		if lanProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "prerouting", "prerouting", "filter", "-150", "{ tcp, udp }", "mark", "set", strconv.Itoa(cfg.FwMark), false, setCN)
		}
		if localProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "output", "output", "route", "-150", "{ tcp, udp }", "mark", "set", strconv.Itoa(cfg.FwMark), true, setCN)
		}
	case "Mix":
		if lanProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "prerouting", "prerouting", "filter", "-150", "udp", "mark", "set", strconv.Itoa(cfg.FwMark), false, setCN)
			cmds = append(cmds,
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting", "meta", "l4proto", "tcp", "mark", "set", strconv.Itoa(cfg.FwMark + 1)}},
				commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", "prerouting_mixtcp", "{", "type", "nat", "hook", "prerouting", "priority", "-100", ";", "}"}},
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "prerouting_mixtcp", "mark", strconv.Itoa(cfg.FwMark + 1), "meta", "l4proto", "tcp", "redirect", "to", cfg.RedirPort}},
			)
		}
		if localProxy {
			cmds = appendNFTRedirChain(cmds, cfg, hosts, "output", "output", "route", "-150", "udp", "mark", "set", strconv.Itoa(cfg.FwMark), true, setCN)
			cmds = append(cmds,
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "output", "meta", "l4proto", "tcp", "mark", "set", strconv.Itoa(cfg.FwMark + 1)}},
				commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", "output_mixtcp", "{", "type", "nat", "hook", "output", "priority", "-100", ";", "}"}},
				commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", "output_mixtcp", "mark", strconv.Itoa(cfg.FwMark + 1), "meta", "l4proto", "tcp", "redirect", "to", cfg.RedirPort}},
			)
		}
	case "T&U旁路转发":
		if area == 5 {
			if lanProxy {
				cmds = appendNFTRedirChain(cmds, cfg, hosts, "prerouting", "prerouting", "filter", "-150", "{ tcp, udp }", "mark", "set", strconv.Itoa(cfg.FwMark), false, setCN)
			}
			if localProxy {
				cmds = appendNFTRedirChain(cmds, cfg, hosts, "output", "output", "route", "-150", "{ tcp, udp }", "mark", "set", strconv.Itoa(cfg.FwMark), true, setCN)
			}
		}
	case "TCP旁路转发":
		if area == 5 {
			if lanProxy {
				cmds = appendNFTRedirChain(cmds, cfg, hosts, "prerouting", "prerouting", "filter", "-150", "tcp", "mark", "set", strconv.Itoa(cfg.FwMark), false, setCN)
			}
			if localProxy {
				cmds = appendNFTRedirChain(cmds, cfg, hosts, "output", "output", "route", "-150", "tcp", "mark", "set", strconv.Itoa(cfg.FwMark), true, setCN)
			}
		}
	}

	return cmds
}

func appendNFTDNSChain(cmds []commandSpec, cfg Config, hosts HostInfo, chain string, hook string) []commandSpec {
	cmds = append(cmds,
		commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "shellcrash", chain, "{", "type", "nat", "hook", hook, "priority", "-100", ";", "}"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "udp", "dport", "!= 53", "return"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "tcp", "dport", "!= 53", "return"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "meta", "mark", strconv.Itoa(cfg.FwMark + 2), "return"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "meta", "skgid", "{", "453,", "7890", "}", "return"}},
	)
	if hook == "prerouting" {
		for _, ip := range hosts.IPv4 {
			cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", "!=", ip, "return"}})
		}
	} else {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", "!=", "127.0.0.0/8", "return"}})
	}
	cmds = append(cmds,
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "udp", "dport", "53", "redirect", "to", cfg.DNSRedirPort}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "tcp", "dport", "53", "redirect", "to", cfg.DNSRedirPort}},
	)
	return cmds
}

func appendNFTRedirChain(cmds []commandSpec, cfg Config, hosts HostInfo, chain string, hook string, chainType string, priority string, jump ...any) []commandSpec {
	chainDef := []string{"add", "chain", "inet", "shellcrash", chain, "{", "type", chainType, "hook", hook, "priority", priority, ";", "}"}
	cmds = append(cmds, commandSpec{Name: "nft", Args: chainDef})
	cmds = append(cmds,
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "tcp", "dport", "53", "return"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "udp", "dport", "53", "return"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "meta", "mark", strconv.Itoa(cfg.FwMark + 2), "return"}},
		commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "meta", "skgid", "7890", "return"}},
	)
	if cfg.FirewallArea == "5" && cfg.BypassHost != "" {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", cfg.BypassHost, "return"}})
	}
	if hook == "prerouting" {
		macList := readFilterList(filepath.Join(cfg.CrashDir, "configs", "mac"))
		ipList := readFilterList(filepath.Join(cfg.CrashDir, "configs", "ip_filter"))
		hostSet := joinForNFTSet(hosts.IPv4)
		switch cfg.MacFilterType {
		case "白名单":
			if len(macList) > 0 && len(ipList) > 0 {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ether", "saddr", "!=", "{", joinForNFTSet(macList), "}", "ip", "saddr", "!=", "{", joinForNFTSet(ipList), "}", "return"}})
			} else if len(macList) > 0 {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ether", "saddr", "!=", "{", joinForNFTSet(macList), "}", "return"}})
			} else if len(ipList) > 0 {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", "!=", "{", joinForNFTSet(ipList), "}", "return"}})
			} else if hostSet != "" {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", "!=", "{", hostSet, "}", "return"}})
			}
		default:
			if len(macList) > 0 {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ether", "saddr", "{", joinForNFTSet(macList), "}", "return"}})
			}
			if len(ipList) > 0 {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", "{", joinForNFTSet(ipList), "}", "return"}})
			}
			if hostSet != "" {
				cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "saddr", "!=", "{", hostSet, "}", "return"}})
			}
		}
	}
	for _, cidr := range hosts.Reserved4 {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "daddr", cidr, "return"}})
	}
	if strings.EqualFold(cfg.IPv6Redir, "ON") && !strings.EqualFold(cfg.CrashCore, "clashpre") {
		for _, cidr := range hosts.Reserved6 {
			cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip6", "daddr", cidr, "return"}})
		}
	} else {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "meta", "nfproto", "ipv6", "return"}})
	}
	if cfg.DNSMode != "fake-ip" && !strings.EqualFold(cfg.CNIPRoute, "OFF") {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "ip", "daddr", "@cn_ip", "return"}})
	}
	if strings.EqualFold(cfg.QuicRJ, "ON") && hook == "prerouting" {
		cmds = append(cmds, commandSpec{Name: "nft", Args: []string{"add", "rule", "inet", "shellcrash", chain, "udp", "dport", "{", "443,", "8443", "}", "return"}})
	}
	jumpArgs := make([]string, 0, len(jump))
	for _, item := range jump {
		switch v := item.(type) {
		case string:
			jumpArgs = append(jumpArgs, v)
		case bool:
		}
	}
	args := []string{"add", "rule", "inet", "shellcrash", chain, "meta", "l4proto"}
	args = append(args, jumpArgs...)
	cmds = append(cmds, commandSpec{Name: "nft", Args: args})
	return cmds
}

func appendNFTCNSets(cmds []commandSpec, cfg Config) []commandSpec {
	v4Path := filepath.Join(cfg.BindDir, "cn_ip.txt")
	v4 := readFilterList(v4Path)
	if len(v4) > 0 {
		cmds = append(cmds,
			commandSpec{Name: "nft", Args: []string{"add", "set", "inet", "shellcrash", "cn_ip", "{", "type", "ipv4_addr", ";", "flags", "interval", ";", "}"}},
			commandSpec{Name: "nft", Args: []string{"add", "element", "inet", "shellcrash", "cn_ip", "{", joinForNFTSet(v4), "}"}},
		)
	}
	v6Path := filepath.Join(cfg.BindDir, "cn_ipv6.txt")
	v6 := readFilterList(v6Path)
	if len(v6) > 0 {
		cmds = append(cmds,
			commandSpec{Name: "nft", Args: []string{"add", "set", "inet", "shellcrash", "cn_ip6", "{", "type", "ipv6_addr", ";", "flags", "interval", ";", "}"}},
			commandSpec{Name: "nft", Args: []string{"add", "element", "inet", "shellcrash", "cn_ip6", "{", joinForNFTSet(v6), "}"}},
		)
	}
	return cmds
}

func readFilterList(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		v := strings.TrimSpace(line)
		if v == "" || strings.HasPrefix(v, "#") {
			continue
		}
		out = append(out, v)
	}
	return dedupeNonEmpty(out)
}

func joinForNFTSet(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ", ")
}

func appendNFTTunForwardAccept(cmds []commandSpec) []commandSpec {
	cmds = append(cmds,
		commandSpec{Name: "nft", Args: []string{"list", "table", "inet", "fw4"}},
		commandSpec{Name: "nft", Args: []string{"add", "table", "inet", "fw4"}},
		commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "fw4", "forward", "{", "type", "filter", "hook", "forward", "priority", "filter", ";", "}"}},
		commandSpec{Name: "nft", Args: []string{"add", "chain", "inet", "fw4", "input", "{", "type", "filter", "hook", "input", "priority", "filter", ";", "}"}},
		commandSpec{Name: "nft", Args: []string{"insert", "rule", "inet", "fw4", "forward", "oifname", "\"utun\"", "accept"}},
		commandSpec{Name: "nft", Args: []string{"insert", "rule", "inet", "fw4", "input", "iifname", "\"utun\"", "accept"}},
	)
	return cmds
}

func hasUTunRoute() bool {
	for i := 0; i < 30; i++ {
		if strings.Contains(string(outputIgnore("ip", "route", "list")), "utun") {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func ensureResolvRepairLine(cfg Config, path string) {
	area := atoiDefault(cfg.FirewallArea, 1)
	if area != 2 && area != 3 {
		return
	}
	if strings.Contains(string(outputIgnore("grep", "127.0.0.1", path)), "127.0.0.1") {
		return
	}
	b := outputIgnore("grep", "-n", "nameserver", path)
	if len(b) == 0 {
		return
	}
	line := strings.TrimSpace(strings.SplitN(string(b), ":", 2)[0])
	if line == "" {
		return
	}
	runIgnore("sed", "-i", line+" i\\nameserver 127.0.0.1 #shellcrash-dns-repair", path)
}

func clearOpenWrtDNSRedirect() {
	out := strings.TrimSpace(string(outputIgnore("uci", "get", "dhcp.@dnsmasq[0].dns_redirect")))
	if out != "1" {
		return
	}
	runIgnore("uci", "del", "dhcp.@dnsmasq[0].dns_redirect")
	runIgnore("uci", "commit", "dhcp.@dnsmasq[0]")
}
