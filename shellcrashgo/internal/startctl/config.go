package startctl

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Config struct {
	CrashDir       string
	BinDir         string
	TmpDir         string
	Command        string
	FirewallArea   string
	FirewallMod    string
	NetworkCheck   string
	RedirMod       string
	DNSMod         string
	DisOverride    string
	DNSPort        string
	RedirPort      string
	TProxyPort     string
	FWMark         string
	RoutingMark    string
	DNSNameServer  string
	DNSFallback    string
	DNSResolver    string
	IPv6DNS        string
	DNSProtect     string
	Authentication string
	ExternalUIURL  string
	HostsOpt       string
	SkipCert       string
	CNIPRoute      string
	IPv6Redir      string
	DBPort         string
	Secret         string
	Host           string
	MixPort        string
	Sniffer        string
	URL            string
	HTTPS          string
	StartOld       string
	StartDelaySec  int
	BotTGService   string
	CrashCore      string
	DebugToFlash   bool
	ShellCrashPID  string
}

func LoadConfig(crashDir string) (Config, error) {
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	envPath := filepath.Join(crashDir, "configs", "command.env")

	cfgKV, err := parseKVFile(cfgPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	envKV, err := parseKVFile(envPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	tmpDir := stripQuotes(envKV["TMPDIR"])
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	binDir := stripQuotes(envKV["BINDIR"])
	if binDir == "" {
		binDir = crashDir
	}

	c := Config{
		CrashDir:       crashDir,
		BinDir:         binDir,
		TmpDir:         tmpDir,
		Command:        stripQuotes(envKV["COMMAND"]),
		FirewallArea:   normalizeONOFFOrRaw(cfgKV["firewall_area"]),
		FirewallMod:    stripQuotes(cfgKV["firewall_mod"]),
		NetworkCheck:   normalizeONOFFOrRaw(cfgKV["network_check"]),
		RedirMod:       stripQuotes(cfgKV["redir_mod"]),
		DNSMod:         stripQuotes(cfgKV["dns_mod"]),
		DisOverride:    stripQuotes(cfgKV["disoverride"]),
		DNSPort:        stripQuotes(cfgKV["dns_port"]),
		RedirPort:      stripQuotes(cfgKV["redir_port"]),
		TProxyPort:     stripQuotes(cfgKV["tproxy_port"]),
		FWMark:         stripQuotes(cfgKV["fwmark"]),
		DNSNameServer:  stripQuotes(cfgKV["dns_nameserver"]),
		DNSFallback:    stripQuotes(cfgKV["dns_fallback"]),
		DNSResolver:    stripQuotes(cfgKV["dns_resolver"]),
		IPv6DNS:        normalizeONOFFOrRaw(cfgKV["ipv6_dns"]),
		DNSProtect:     normalizeONOFFOrRaw(cfgKV["dns_protect"]),
		Authentication: stripQuotes(cfgKV["authentication"]),
		ExternalUIURL:  stripQuotes(cfgKV["external_ui_url"]),
		HostsOpt:       normalizeONOFFOrRaw(cfgKV["hosts_opt"]),
		SkipCert:       normalizeONOFFOrRaw(cfgKV["skip_cert"]),
		CNIPRoute:      normalizeONOFFOrRaw(cfgKV["cn_ip_route"]),
		IPv6Redir:      normalizeONOFFOrRaw(cfgKV["ipv6_redir"]),
		DBPort:         stripQuotes(cfgKV["db_port"]),
		Secret:         stripQuotes(cfgKV["secret"]),
		Host:           stripQuotes(cfgKV["host"]),
		MixPort:        stripQuotes(cfgKV["mix_port"]),
		Sniffer:        normalizeONOFFOrRaw(cfgKV["sniffer"]),
		URL:            stripQuotes(cfgKV["Url"]),
		HTTPS:          stripQuotes(cfgKV["Https"]),
		StartOld:       normalizeONOFFOrRaw(cfgKV["start_old"]),
		BotTGService:   normalizeONOFFOrRaw(cfgKV["bot_tg_service"]),
		CrashCore:      stripQuotes(cfgKV["crashcore"]),
	}
	if n, err := strconv.Atoi(stripQuotes(cfgKV["start_delay"])); err == nil {
		c.StartDelaySec = n
	}

	if c.FirewallArea == "" {
		c.FirewallArea = "1"
	}
	if c.DBPort == "" {
		c.DBPort = "9999"
	}
	if c.StartOld == "" {
		c.StartOld = "OFF"
	}
	if c.FirewallMod == "" {
		c.FirewallMod = "iptables"
	}
	if c.CNIPRoute == "" {
		c.CNIPRoute = "ON"
	}
	if c.HostsOpt == "" {
		c.HostsOpt = "ON"
	}
	if c.SkipCert == "" {
		c.SkipCert = "ON"
	}
	if c.IPv6Redir == "" {
		c.IPv6Redir = "OFF"
	}
	if c.BotTGService == "" {
		c.BotTGService = "OFF"
	}
	if c.CrashCore == "" {
		c.CrashCore = "meta"
	}
	if c.MixPort == "" {
		c.MixPort = "7890"
	}
	if c.RedirPort == "" {
		c.RedirPort = "7892"
	}
	if c.TProxyPort == "" {
		c.TProxyPort = "7893"
	}
	if c.DNSPort == "" {
		c.DNSPort = "1053"
	}
	if c.FWMark == "" {
		c.FWMark = c.RedirPort
	}
	if n, err := strconv.Atoi(c.FWMark); err == nil {
		c.RoutingMark = strconv.Itoa(n + 2)
	} else {
		c.RoutingMark = "7894"
	}
	if c.DNSNameServer == "" {
		c.DNSNameServer = "223.5.5.5, 1.2.4.8"
	}
	if c.DNSFallback == "" {
		c.DNSFallback = "1.1.1.1, 8.8.8.8"
	}
	if c.DNSResolver == "" {
		c.DNSResolver = "223.5.5.5, 2400:3200::1"
	}
	if c.DNSProtect == "" {
		c.DNSProtect = "ON"
	}
	if c.IPv6DNS == "" {
		c.IPv6DNS = "ON"
	}
	if c.Sniffer == "" {
		c.Sniffer = "OFF"
	}
	if c.Command == "" {
		if strings.Contains(c.CrashCore, "singbox") {
			c.Command = "$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons"
		} else {
			c.Command = "$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"
		}
	}

	if b, err := os.ReadFile(filepath.Join(tmpDir, "shellcrash.pid")); err == nil {
		c.ShellCrashPID = strings.TrimSpace(string(b))
	}

	return c, nil
}

func defaultCommandForCore(crashCore string) string {
	if strings.Contains(crashCore, "singbox") {
		return "$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons"
	}
	return "$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"
}

func parseKVFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func writeKVFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func normalizeONOFFOrRaw(s string) string {
	s = stripQuotes(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	u := strings.ToUpper(s)
	if u == "ON" || u == "OFF" {
		return u
	}
	if _, err := strconv.Atoi(s); err == nil {
		return s
	}
	return s
}
