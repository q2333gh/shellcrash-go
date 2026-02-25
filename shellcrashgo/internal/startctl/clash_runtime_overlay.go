package startctl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var clashMainSections = []string{
	"proxies",
	"proxy-groups",
	"proxy-providers",
	"rules",
	"rule-providers",
	"sub-rules",
	"listeners",
}

var clashSkipCertPattern = regexp.MustCompile(`(?m)(skip-cert-verify:\s*)(true|false)`)

func (c *Controller) buildClashRuntimeConfigFromCore(core string, includeCustom bool) ([]byte, error) {
	sections := extractTopSections(core, clashMainSections)
	userYAML := ""
	othersYAML := ""
	customProxies := ""
	customProxyGroups := ""
	customRules := ""

	if includeCustom {
		userYAML = c.readOptionalYAML(filepath.Join(c.Cfg.CrashDir, "yamls", "user.yaml"))
		othersYAML = c.readOptionalYAML(filepath.Join(c.Cfg.CrashDir, "yamls", "others.yaml"))
		customProxies = c.readOptionalYAML(filepath.Join(c.Cfg.CrashDir, "yamls", "proxies.yaml"))
		customProxyGroups = c.readOptionalYAML(filepath.Join(c.Cfg.CrashDir, "yamls", "proxy-groups.yaml"))
		customRules = c.readOptionalYAML(filepath.Join(c.Cfg.CrashDir, "yamls", "rules.yaml"))
		if strings.TrimSpace(customProxies) != "" {
			sections["proxies"] = prependYAMLListItems("proxies", sections["proxies"], customProxies)
		}
		if strings.TrimSpace(customProxyGroups) != "" {
			sections["proxy-groups"] = prependYAMLListItems("proxy-groups", sections["proxy-groups"], customProxyGroups)
		}
		if strings.TrimSpace(customRules) != "" {
			sections["rules"] = prependYAMLListItems("rules", sections["rules"], customRules)
		}
	}

	var out strings.Builder
	out.WriteString(c.clashSetYAML(userYAML))
	out.WriteByte('\n')

	if !hasTopSection(core, "dns") {
		out.WriteString(c.clashDNSYAML())
		out.WriteByte('\n')
	}
	if c.shouldEmitHostsYAML(userYAML) {
		out.WriteString(c.clashHostsYAML())
		out.WriteByte('\n')
	}
	if userYAML != "" {
		out.WriteString(userYAML)
		out.WriteByte('\n')
	}
	if othersYAML != "" {
		out.WriteString(othersYAML)
		out.WriteByte('\n')
	}

	for _, name := range clashMainSections {
		sec := strings.TrimSpace(sections[name])
		if sec == "" {
			continue
		}
		out.WriteString(trimLeadingNewline(sec))
		if !strings.HasSuffix(sec, "\n") {
			out.WriteByte('\n')
		}
		out.WriteByte('\n')
	}

	if isMode(c.Cfg.DNSMod, "mix", "route") {
		if sec := sections["rule-providers"]; strings.TrimSpace(sec) == "" || !strings.Contains(sec, "cn:") {
			if !strings.Contains(othersYAML, "rule-providers:") {
				out.WriteString("rule-providers:\n")
				out.WriteString("  cn: {type: http, behavior: domain, format: mrs, path: ./ruleset/cn.mrs, url: https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@update/bin/geodata/mrs_geosite_cn.mrs}\n\n")
			}
		}
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		return nil, fmt.Errorf("generated empty clash runtime config")
	}
	text = c.applyClashSkipCert(text)
	return []byte(text + "\n"), nil
}

func (c *Controller) applyClashSkipCert(text string) string {
	want := "true"
	if strings.EqualFold(c.Cfg.SkipCert, "OFF") {
		want = "false"
	}
	return clashSkipCertPattern.ReplaceAllString(text, "${1}"+want)
}

func (c *Controller) clashSetYAML(userYAML string) string {
	auth := strings.TrimSpace(c.Cfg.Authentication)
	tun := "tun: {enable: false}"
	if isMode(c.Cfg.RedirMod, "mix", "tun") {
		if c.Cfg.CrashCore == "meta" {
			tun = "tun: {enable: true, stack: system, device: utun, auto-route: false, auto-detect-interface: false}"
		} else {
			tun = "tun: {enable: true, stack: system}"
		}
	}
	experimental := "experimental: {ignore-resolve-fail: true, interface-name: en0}"
	if c.Cfg.CrashCore == "clashpre" && (strings.EqualFold(c.Cfg.DNSMod, "redir_host") || strings.EqualFold(c.Cfg.Sniffer, "ON")) {
		experimental = "experimental: {ignore-resolve-fail: true, interface-name: en0, sniff-tls-sni: true}"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "mixed-port: %d\n", mustInt(c.Cfg.MixPort, 7890))
	fmt.Fprintf(&b, "redir-port: %d\n", mustInt(c.Cfg.RedirPort, 7892))
	fmt.Fprintf(&b, "tproxy-port: %d\n", mustInt(c.Cfg.TProxyPort, 7893))
	fmt.Fprintf(&b, "authentication: [\"%s\"]\n", escapeYAMLDoubleQuoted(auth))
	b.WriteString("allow-lan: true\n")
	b.WriteString("mode: Rule\n")
	b.WriteString("log-level: info\n")
	b.WriteString("ipv6: true\n")
	fmt.Fprintf(&b, "external-controller: :%s\n", valueOr(c.Cfg.DBPort, "9999"))
	b.WriteString("external-ui: ui\n")
	fmt.Fprintf(&b, "external-ui-url: \"%s\"\n", escapeYAMLDoubleQuoted(c.Cfg.ExternalUIURL))
	fmt.Fprintf(&b, "secret: %s\n", strings.TrimSpace(c.Cfg.Secret))
	b.WriteString(tun + "\n")
	b.WriteString(experimental + "\n")
	if strings.EqualFold(c.Cfg.Sniffer, "ON") && c.Cfg.CrashCore == "meta" {
		b.WriteString("sniffer: {enable: true, parse-pure-ip: true, skip-domain: [Mijia Cloud], sniff: {http: {ports: [80, 8080-8880], override-destination: true}, tls: {ports: [443, 8443]}, quic: {ports: [443, 8443]}}}\n")
	}
	fmt.Fprintf(&b, "routing-mark: %d\n", mustInt(c.Cfg.RoutingMark, 7894))
	b.WriteString("unified-delay: true\n")
	return removeOverriddenSetKeys(b.String(), userYAML)
}

func (c *Controller) clashDNSYAML() string {
	ipv6 := "true"
	if strings.EqualFold(c.Cfg.IPv6DNS, "OFF") {
		ipv6 = "false"
	}
	resolver := splitCSV(c.Cfg.DNSResolver)
	if c.Cfg.CrashCore != "meta" {
		resolver = []string{"223.5.5.5"}
	}
	direct := splitCSV(c.Cfg.DNSNameServer)
	filter := []string{"+.*"}
	if isMode(c.Cfg.DNSMod, "mix", "fake-ip") {
		if loaded := c.loadFakeIPFilter(); len(loaded) > 0 {
			filter = loaded
		}
		if strings.EqualFold(c.Cfg.DNSMod, "mix") {
			filter = append(filter, "rule-set:cn")
		}
	}

	var b strings.Builder
	b.WriteString("dns:\n")
	b.WriteString("  enable: true\n")
	fmt.Fprintf(&b, "  listen: :%d\n", mustInt(c.Cfg.DNSPort, 1053))
	b.WriteString("  use-hosts: true\n")
	fmt.Fprintf(&b, "  ipv6: %s\n", ipv6)
	fmt.Fprintf(&b, "  default-nameserver: [ %s ]\n", joinCSVForYAML(resolver))
	fmt.Fprintf(&b, "  direct-nameserver: [ %s ]\n", joinCSVForYAML(direct))
	b.WriteString("  enhanced-mode: fake-ip\n")
	b.WriteString("  fake-ip-range: 28.0.0.0/8\n")
	b.WriteString("  fake-ip-range6: fc00::/16\n")
	b.WriteString("  fake-ip-filter:\n")
	for _, line := range filter {
		fmt.Fprintf(&b, "    - '%s'\n", strings.ReplaceAll(line, "'", "''"))
	}

	if isMode(c.Cfg.DNSMod, "mix", "route") {
		final := splitCSV(c.Cfg.DNSFallback)
		if strings.EqualFold(c.Cfg.DNSProtect, "OFF") {
			final = direct
		}
		b.WriteString("  respect-rules: true\n")
		fmt.Fprintf(&b, "  nameserver-policy: {'rule-set:cn': [ %s ]}\n", joinCSVForYAML(direct))
		fmt.Fprintf(&b, "  proxy-server-nameserver : [ %s ]\n", joinCSVForYAML(resolver))
		fmt.Fprintf(&b, "  nameserver: [ %s ]\n", joinCSVForYAML(final))
	} else {
		fmt.Fprintf(&b, "  nameserver: [ %s ]\n", joinCSVForYAML(direct))
	}
	return b.String()
}

func (c *Controller) shouldEmitHostsYAML(userYAML string) bool {
	if strings.EqualFold(c.Cfg.HostsOpt, "OFF") {
		return false
	}
	return !regexp.MustCompile(`(?m)^\s*hosts\s*:`).MatchString(userYAML)
}

func (c *Controller) clashHostsYAML() string {
	entries := map[string]string{
		"time.android.com":  "203.107.6.88",
		"time.facebook.com": "203.107.6.88",
	}
	if c.Cfg.CrashCore == "meta" {
		entries["services.googleapis.cn"] = "services.googleapis.com"
	}
	for _, e := range c.readSystemHostEntries("/etc/hosts", "/data/etc/custom_hosts") {
		if _, exists := entries[e.Domain]; exists {
			continue
		}
		entries[e.Domain] = e.IP
	}

	ordered := []string{
		"services.googleapis.cn",
		"time.android.com",
		"time.facebook.com",
	}
	seen := map[string]struct{}{}
	var b strings.Builder
	b.WriteString("use-system-hosts: true\n")
	b.WriteString("hosts:\n")
	for _, domain := range ordered {
		ip, ok := entries[domain]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "  '%s': %s\n", domain, ip)
		seen[domain] = struct{}{}
	}
	dyn := make([]string, 0, len(entries))
	for domain := range entries {
		if _, ok := seen[domain]; ok {
			continue
		}
		dyn = append(dyn, domain)
	}
	sort.Strings(dyn)
	for _, domain := range dyn {
		fmt.Fprintf(&b, "  '%s': %s\n", domain, entries[domain])
	}
	return b.String()
}

type hostEntry struct {
	IP     string
	Domain string
}

func (c *Controller) readSystemHostEntries(paths ...string) []hostEntry {
	var out []hostEntry
	seen := map[string]struct{}{}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.Index(line, "#"); idx >= 0 {
				line = strings.TrimSpace(line[:idx])
			}
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			ip := fields[0]
			domain := fields[1]
			if !isIPv4(ip) {
				continue
			}
			key := domain + "\x00" + ip
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, hostEntry{IP: ip, Domain: domain})
		}
		_ = f.Close()
	}
	return out
}

func isIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		n := 0
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return false
			}
			n = n*10 + int(ch-'0')
		}
		if n < 0 || n > 255 {
			return false
		}
	}
	return true
}

func (c *Controller) loadFakeIPFilter() []string {
	paths := []string{
		filepath.Join(c.Cfg.CrashDir, "configs", "fake_ip_filter"),
		filepath.Join(c.Cfg.CrashDir, "configs", "fake_ip_filter.list"),
	}
	out := make([]string, 0, 64)
	seen := map[string]struct{}{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "Mijia") {
				continue
			}
			if _, ok := seen[line]; ok {
				continue
			}
			seen[line] = struct{}{}
			out = append(out, line)
		}
	}
	return out
}

func (c *Controller) readOptionalYAML(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func removeOverriddenSetKeys(setYAML, userYAML string) string {
	if strings.TrimSpace(userYAML) == "" {
		return setYAML
	}
	keys := []string{
		"mode", "allow-lan", "log-level", "tun", "experimental", "external-ui-url",
		"interface-name", "dns", "store-selected", "unified-delay",
	}
	userHas := map[string]bool{}
	userLines := strings.Split(userYAML, "\n")
	for _, line := range userLines {
		trimmed := strings.TrimSpace(line)
		for _, key := range keys {
			if strings.HasPrefix(trimmed, key+":") {
				userHas[key] = true
			}
		}
	}
	if len(userHas) == 0 {
		return setYAML
	}
	setLines := strings.Split(setYAML, "\n")
	out := make([]string, 0, len(setLines))
	for _, line := range setLines {
		trimmed := strings.TrimSpace(line)
		skip := false
		for key := range userHas {
			if strings.HasPrefix(trimmed, key+":") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func prependYAMLListItems(name, section, custom string) string {
	if strings.TrimSpace(custom) == "" {
		return section
	}
	baseItems := extractYAMLListItems(section)
	customItems := extractYAMLListItems(custom)
	merged := append(customItems, baseItems...)
	return strings.TrimRight(renderYAMLListSection(name, merged), "\n")
}

func extractYAMLListItems(section string) []string {
	lines := strings.Split(section, "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			items = append(items, strings.TrimPrefix(trimmed, "- "))
		}
	}
	return dedupe(items)
}

func renderYAMLListSection(name string, items []string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(name + ":\n")
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		b.WriteString("  - " + item + "\n")
	}
	return b.String()
}

func dedupe(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinCSVForYAML(items []string) string {
	if len(items) == 0 {
		return "223.5.5.5"
	}
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, "'"+strings.ReplaceAll(item, "'", "''")+"'")
	}
	return strings.Join(quoted, ", ")
}

func extractTopSections(text string, sectionNames []string) map[string]string {
	want := map[string]struct{}{}
	for _, s := range sectionNames {
		want[s] = struct{}{}
	}
	out := map[string]string{}
	lines := strings.Split(text, "\n")
	current := ""
	var buf strings.Builder
	flush := func() {
		if current != "" {
			out[current] = strings.TrimRight(buf.String(), "\n")
		}
		buf.Reset()
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			if current != "" {
				buf.WriteString(line + "\n")
			}
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || trimmed == "" {
			if current != "" {
				buf.WriteString(line + "\n")
			}
			continue
		}
		name := strings.TrimSuffix(strings.SplitN(trimmed, ":", 2)[0], " ")
		if _, ok := want[name]; ok && strings.HasPrefix(trimmed, name+":") {
			flush()
			current = name
			buf.WriteString(line + "\n")
			continue
		}
		flush()
		current = ""
	}
	flush()
	return out
}

func hasTopSection(text, name string) bool {
	lines := strings.Split(text, "\n")
	prefix := name + ":"
	for _, line := range lines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func trimLeadingNewline(s string) string {
	return strings.TrimPrefix(s, "\n")
}

func escapeYAMLDoubleQuoted(v string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(v)
}
