package firewall

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	CrashDir      string
	BindDir       string
	CommonPorts   string
	MultiPort     string
	DNSMode       string
	CNIPRoute     string
	FwMark        int
	Table         int
	TProxyPort    string
	RedirPort     string
	DNSRedirPort  string
	MixPort       string
	DBPort        string
	FWWanPorts    string
	VMSPort       string
	SSSPort       string
	FirewallArea  string
	FirewallMod   string
	RedirMod      string
	IPv6Redir     string
	BypassHost    string
	FWWan         string
	MacFilterType string
	CrashCore     string
	CustHostIPv4  string
	ReplaceHostV4 string
	ReserveIPv4   string
	ReserveIPv6   string
	TSService     string
	WGService     string
	WGIPv4        string
	WGIPv6        string
	VMRedir       string
	VMIPv4        string
	QuicRJ        string
}

func LoadConfig(crashDir string) (Config, error) {
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	gwPath := filepath.Join(crashDir, "configs", "gateway.cfg")
	envPath := filepath.Join(crashDir, "configs", "command.env")

	cfgKV, err := parseKVFile(cfgPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	gwKV, err := parseKVFile(gwPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	envKV, err := parseKVFile(envPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	redirPort := atoiDefault(stripQuotes(cfgKV["redir_port"]), 7892)
	c := Config{
		CrashDir:      crashDir,
		BindDir:       defaultString(stripQuotes(envKV["BINDIR"]), crashDir),
		CommonPorts:   defaultString(normalizeONOFFOrRaw(cfgKV["common_ports"]), "ON"),
		MultiPort:     defaultString(stripQuotes(cfgKV["multiport"]), "22,80,443,8080,8443"),
		DNSMode:       defaultString(stripQuotes(cfgKV["dns_mod"]), "redir_host"),
		CNIPRoute:     defaultString(normalizeONOFFOrRaw(cfgKV["cn_ip_route"]), "ON"),
		FwMark:        atoiDefault(stripQuotes(cfgKV["fwmark"]), redirPort),
		Table:         atoiDefault(stripQuotes(cfgKV["table"]), 100),
		TProxyPort:    defaultString(stripQuotes(cfgKV["tproxy_port"]), "7893"),
		RedirPort:     defaultString(stripQuotes(cfgKV["redir_port"]), "7892"),
		DNSRedirPort:  defaultString(stripQuotes(cfgKV["dns_redir_port"]), "1053"),
		MixPort:       defaultString(stripQuotes(cfgKV["mix_port"]), "7890"),
		DBPort:        defaultString(stripQuotes(cfgKV["db_port"]), "9999"),
		FWWanPorts:    stripQuotes(gwKV["fw_wan_ports"]),
		VMSPort:       stripQuotes(gwKV["vms_port"]),
		SSSPort:       stripQuotes(gwKV["sss_port"]),
		FirewallArea:  defaultString(stripQuotes(cfgKV["firewall_area"]), "1"),
		FirewallMod:   defaultString(stripQuotes(cfgKV["firewall_mod"]), "iptables"),
		RedirMod:      stripQuotes(cfgKV["redir_mod"]),
		IPv6Redir:     defaultString(normalizeONOFFOrRaw(cfgKV["ipv6_redir"]), "OFF"),
		BypassHost:    stripQuotes(cfgKV["bypass_host"]),
		FWWan:         defaultString(normalizeONOFFOrRaw(cfgKV["fw_wan"]), "ON"),
		MacFilterType: defaultString(stripQuotes(cfgKV["macfilter_type"]), "黑名单"),
		CrashCore:     defaultString(stripQuotes(cfgKV["crashcore"]), "meta"),
		CustHostIPv4:  stripQuotes(cfgKV["cust_host_ipv4"]),
		ReplaceHostV4: defaultString(normalizeONOFFOrRaw(cfgKV["replace_default_host_ipv4"]), "OFF"),
		ReserveIPv4:   stripQuotes(cfgKV["reserve_ipv4"]),
		ReserveIPv6:   stripQuotes(cfgKV["reserve_ipv6"]),
		TSService:     defaultString(normalizeONOFFOrRaw(cfgKV["ts_service"]), "OFF"),
		WGService:     defaultString(normalizeONOFFOrRaw(cfgKV["wg_service"]), "OFF"),
		WGIPv4:        stripQuotes(gwKV["wg_ipv4"]),
		WGIPv6:        stripQuotes(gwKV["wg_ipv6"]),
		VMRedir:       defaultString(normalizeONOFFOrRaw(cfgKV["vm_redir"]), "OFF"),
		VMIPv4:        stripQuotes(cfgKV["vm_ipv4"]),
		QuicRJ:        defaultString(normalizeONOFFOrRaw(cfgKV["quic_rj"]), "OFF"),
	}
	if c.RedirMod == "" {
		c.RedirMod = "Redir"
	}
	return c, nil
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
	s = stripQuotes(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	u := strings.ToUpper(s)
	if u == "ON" || u == "OFF" {
		return u
	}
	return s
}

func defaultString(v string, def string) string {
	if v == "" {
		return def
	}
	return v
}

func atoiDefault(v string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}
