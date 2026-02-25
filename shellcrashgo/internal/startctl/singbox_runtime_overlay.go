package startctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type sbOutbound struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

type sbParsedServer struct {
	Type string
	Host string
	Port int
}

func (c *Controller) writeSingboxRuntimeOverlays(jsonDir, baseConfig string) error {
	if err := writePrettyJSON(filepath.Join(jsonDir, "10_log.json"), map[string]any{
		"log": map[string]any{"level": "info", "timestamp": true},
	}); err != nil {
		return err
	}
	if err := writePrettyJSON(filepath.Join(jsonDir, "20_experimental.json"), map[string]any{
		"experimental": map[string]any{
			"clash_api": map[string]any{
				"external_controller":      "0.0.0.0:" + c.Cfg.DBPort,
				"external_ui":              "ui",
				"external_ui_download_url": c.Cfg.ExternalUIURL,
				"secret":                   c.Cfg.Secret,
				"default_mode":             "Rule",
			},
		},
	}); err != nil {
		return err
	}
	if err := c.writeSingboxInbounds(filepath.Join(jsonDir, "30_inbounds.json")); err != nil {
		return err
	}
	if err := c.writeSingboxHosts(filepath.Join(jsonDir, "35_add_hosts.json")); err != nil {
		return err
	}
	if err := c.writeSingboxDNS(filepath.Join(jsonDir, "40_dns.json")); err != nil {
		return err
	}
	if err := c.writeSingboxRoute(filepath.Join(jsonDir, "45_add_route.json")); err != nil {
		return err
	}
	if err := c.writeSingboxOutbounds(filepath.Join(jsonDir, "46_add_outbounds.json"), baseConfig); err != nil {
		return err
	}
	if isMode(c.Cfg.DNSMod, "mix", "route") {
		if err := writePrettyJSON(filepath.Join(jsonDir, "47_add_rule_set.json"), map[string]any{
			"route": map[string]any{
				"rule_set": []map[string]any{
					{
						"tag":             "cn",
						"type":            "remote",
						"format":          "binary",
						"path":            "./ruleset/cn.srs",
						"url":             "https://testingcf.jsdelivr.net/gh/DustinWin/ruleset_geodata@sing-box-ruleset/cn.srs",
						"download_detour": "DIRECT",
					},
				},
			},
		}); err != nil {
			return err
		}
	}
	if strings.EqualFold(c.Cfg.RedirMod, "Mix") || strings.EqualFold(c.Cfg.RedirMod, "Tun") {
		address := []string{"28.0.0.1/30"}
		if strings.EqualFold(c.Cfg.IPv6Redir, "ON") {
			address = append([]string{"fe80::e5c5:2469:d09b:609a/64"}, address...)
		}
		if err := writePrettyJSON(filepath.Join(jsonDir, "48_tun.json"), map[string]any{
			"inbounds": []map[string]any{
				{
					"type":           "tun",
					"tag":            "tun-in",
					"interface_name": "utun",
					"address":        address,
					"auto_route":     false,
					"stack":          "system",
				},
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) writeSingboxHosts(path string) error {
	if strings.EqualFold(c.Cfg.HostsOpt, "OFF") {
		return nil
	}
	hostPaths := []string{filepath.Join(os.Getenv("HOME"), ".hosts"), "/etc/hosts"}
	if _, err := os.Stat("/data/etc/custom_hosts"); err == nil {
		hostPaths = append([]string{"/data/etc/custom_hosts"}, hostPaths...)
	}
	return writePrettyJSON(path, map[string]any{
		"dns": map[string]any{
			"servers": []map[string]any{
				{
					"type": "hosts",
					"tag":  "hosts",
					"path": hostPaths,
					"predefined": map[string]any{
						"localhost":         []string{"127.0.0.1", "::1"},
						"time.android.com":  "203.107.6.88",
						"time.facebook.com": "203.107.6.88",
					},
				},
			},
			"rules": []map[string]any{
				{
					"ip_accept_any": true,
					"server":        "hosts",
				},
			},
		},
	})
}

func (c *Controller) writeSingboxInbounds(path string) error {
	mixed := map[string]any{
		"type":        "mixed",
		"tag":         "mixed-in",
		"listen":      "::",
		"listen_port": mustInt(c.Cfg.MixPort, 7890),
	}
	if user, pass, ok := parseAuth(c.Cfg.Authentication); ok {
		mixed["users"] = []map[string]string{{"username": user, "password": pass}}
	}
	inbounds := []map[string]any{
		mixed,
		{"type": "direct", "tag": "dns-in", "listen": "::", "listen_port": mustInt(c.Cfg.DNSPort, 1053)},
		{"type": "redirect", "tag": "redirect-in", "listen": "::", "listen_port": mustInt(c.Cfg.RedirPort, 7892)},
		{"type": "tproxy", "tag": "tproxy-in", "listen": "::", "listen_port": mustInt(c.Cfg.TProxyPort, 7893)},
	}
	return writePrettyJSON(path, map[string]any{"inbounds": inbounds})
}

func (c *Controller) writeSingboxDNS(path string) error {
	strategy := "prefer_ipv4"
	if strings.EqualFold(c.Cfg.IPv6DNS, "OFF") {
		strategy = "ipv4_only"
	}

	globalTag := "dns_proxy"
	directRule := map[string]any{"inbound": []string{"dns-in"}, "server": "dns_direct"}
	proxyRule := map[string]any{"query_type": []string{"A", "AAAA"}, "server": "dns_proxy", "strategy": strategy, "rewrite_ttl": 1}
	switch strings.ToLower(c.Cfg.DNSMod) {
	case "fake-ip", "mix":
		globalTag = "dns_fakeip"
		proxyRule["server"] = "dns_fakeip"
		if strings.EqualFold(c.Cfg.DNSMod, "mix") {
			directRule = map[string]any{"rule_set": []string{"cn"}, "server": "dns_direct"}
		}
	case "route":
		directRule = map[string]any{"rule_set": []string{"cn"}, "server": "dns_direct"}
	}

	finalDNS := "dns_proxy"
	if strings.EqualFold(c.Cfg.DNSProtect, "OFF") {
		finalDNS = "dns_direct"
	}

	servers := []map[string]any{
		serverEntry("dns_proxy", parseSBServer(c.Cfg.DNSFallback), c.Cfg.RoutingMark, "dns_resolver"),
		serverEntry("dns_direct", parseSBServer(c.Cfg.DNSNameServer), c.Cfg.RoutingMark, "dns_resolver"),
		{
			"tag":         "dns_fakeip",
			"type":        "fakeip",
			"inet4_range": "28.0.0.0/8",
			"inet6_range": "fc00::/16",
		},
		serverEntry("dns_resolver", parseSBServer(c.Cfg.DNSResolver), c.Cfg.RoutingMark, ""),
	}
	rules := []map[string]any{
		{"clash_mode": "Direct", "server": "dns_direct", "strategy": strategy},
		{"domain_suffix": []string{"services.googleapis.cn"}, "server": "dns_fakeip", "strategy": strategy, "rewrite_ttl": 1},
	}
	if isMode(c.Cfg.DNSMod, "fake-ip", "mix") {
		rules = append(rules, c.buildSingboxFakeIPFilterRules()...)
	}
	rules = append(rules,
		map[string]any{"clash_mode": "Global", "query_type": []string{"A", "AAAA"}, "server": globalTag, "strategy": strategy, "rewrite_ttl": 1},
		directRule,
		proxyRule,
	)
	return writePrettyJSON(path, map[string]any{
		"dns": map[string]any{
			"servers":           servers,
			"rules":             rules,
			"final":             finalDNS,
			"strategy":          strategy,
			"independent_cache": true,
			"reverse_mapping":   true,
		},
	})
}

func (c *Controller) buildSingboxFakeIPFilterRules() []map[string]any {
	lines := c.loadFakeIPFilter()
	domains := make([]string, 0, len(lines))
	suffixes := make([]string, 0, len(lines))
	regexes := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, ".*") {
			regex := strings.ReplaceAll(line, ".", `\.`)
			regex = strings.ReplaceAll(regex, "*", ".*")
			if strings.HasPrefix(regex, "+") {
				regex = `.+` + regex[1:]
			}
			regexes = append(regexes, regex)
			continue
		}
		if strings.HasPrefix(line, "*") || strings.HasPrefix(line, "+") {
			suffix := strings.TrimPrefix(strings.TrimPrefix(line, "*."), "+.")
			if suffix == line {
				suffix = strings.TrimPrefix(strings.TrimPrefix(line, "*"), "+")
			}
			suffix = strings.TrimSpace(suffix)
			if suffix != "" {
				suffixes = append(suffixes, suffix)
			}
			continue
		}
		domains = append(domains, line)
	}

	out := make([]map[string]any, 0, 3)
	if len(domains) > 0 {
		out = append(out, map[string]any{"domain": domains, "server": "dns_direct"})
	}
	if len(suffixes) > 0 {
		out = append(out, map[string]any{"domain_suffix": suffixes, "server": "dns_direct"})
	}
	if len(regexes) > 0 {
		out = append(out, map[string]any{"domain_regex": regexes, "server": "dns_direct"})
	}
	return out
}

func (c *Controller) writeSingboxRoute(path string) error {
	rules := []map[string]any{
		{"inbound": []string{"dns-in"}, "action": "hijack-dns"},
		{"clash_mode": "Direct", "outbound": "DIRECT"},
		{"clash_mode": "Global", "outbound": "GLOBAL"},
	}
	if strings.EqualFold(c.Cfg.Sniffer, "ON") {
		rules = append([]map[string]any{{"action": "sniff", "timeout": "500ms"}}, rules...)
	}
	return writePrettyJSON(path, map[string]any{
		"route": map[string]any{
			"default_domain_resolver": "dns_resolver",
			"default_mark":            mustInt(c.Cfg.RoutingMark, 7894),
			"rules":                   rules,
		},
	})
}

func (c *Controller) writeSingboxOutbounds(path, baseConfig string) error {
	base := struct {
		Outbounds []sbOutbound `json:"outbounds"`
	}{}
	raw, err := os.ReadFile(baseConfig)
	if err == nil {
		_ = json.Unmarshal(raw, &base)
	}

	hasTag := map[string]bool{}
	globalCandidates := []string{}
	for _, ob := range base.Outbounds {
		tag := strings.TrimSpace(ob.Tag)
		if tag == "" {
			continue
		}
		hasTag[tag] = true
		if ob.Type == "selector" || ob.Type == "urltest" {
			globalCandidates = append(globalCandidates, tag)
		}
	}

	add := []map[string]any{}
	if !hasTag["DIRECT"] {
		add = append(add, map[string]any{"tag": "DIRECT", "type": "direct"})
	}
	if !hasTag["REJECT"] {
		add = append(add, map[string]any{"tag": "REJECT", "type": "block"})
	}
	if !hasTag["GLOBAL"] && len(globalCandidates) > 0 {
		global := append([]string{}, globalCandidates...)
		global = append(global, "DIRECT")
		add = append(add, map[string]any{"tag": "GLOBAL", "type": "selector", "outbounds": global})
	}
	if len(add) == 0 {
		return nil
	}
	return writePrettyJSON(path, map[string]any{"outbounds": add})
}

func serverEntry(tag string, parsed sbParsedServer, mark string, resolver string) map[string]any {
	out := map[string]any{
		"tag":          tag,
		"type":         parsed.Type,
		"server":       parsed.Host,
		"server_port":  parsed.Port,
		"routing_mark": mustInt(mark, 7894),
	}
	if resolver != "" {
		out["domain_resolver"] = resolver
	}
	return out
}

func writePrettyJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func parseSBServer(raw string) sbParsedServer {
	first := strings.TrimSpace(strings.Split(raw, ",")[0])
	if first == "" {
		return sbParsedServer{Type: "udp", Host: "223.5.5.5", Port: 53}
	}
	proto := "udp"
	rest := first
	if i := strings.Index(first, "://"); i > 0 {
		proto = strings.ToLower(first[:i])
		rest = first[i+3:]
	}
	host := rest
	port := ""
	if strings.HasPrefix(rest, "[") {
		if i := strings.Index(rest, "]"); i > 1 {
			host = rest[1:i]
			if i+1 < len(rest) && rest[i+1] == ':' {
				port = rest[i+2:]
			}
		}
	} else {
		if j := strings.IndexAny(rest, ":/"); j >= 0 {
			host = rest[:j]
			if rest[j] == ':' {
				port = rest[j+1:]
			}
		}
	}
	if host == "" {
		host = "223.5.5.5"
	}
	p := defaultDNSPort(proto)
	if n, err := strconv.Atoi(port); err == nil && n > 0 {
		p = n
	}
	return sbParsedServer{Type: proto, Host: host, Port: p}
}

func defaultDNSPort(proto string) int {
	switch proto {
	case "doh", "https":
		return 443
	case "dot", "tls":
		return 853
	default:
		return 53
	}
}

func parseAuth(s string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func mustInt(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func isMode(v string, vals ...string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, want := range vals {
		if v == strings.ToLower(want) {
			return true
		}
	}
	return false
}
