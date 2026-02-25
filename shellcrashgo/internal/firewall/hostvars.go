package firewall

import (
	"os"
	"regexp"
	"strings"
)

type HostVars struct {
	HostIPv4    string
	HostIPv6    string
	LocalIPv4   string
	ReserveIPv4 string
	ReserveIPv6 string
}

var localSrcIPv4Pattern = regexp.MustCompile(`\bsrc\s+((?:\d{1,3}\.){3}\d{1,3})\b`)

func GetHostVars(crashDir string, env map[string]string) (HostVars, error) {
	cfg, err := LoadConfig(crashDir)
	if err != nil {
		return HostVars{}, err
	}
	applyHostVarEnvOverrides(&cfg, env)

	hosts := detectHosts(cfg)
	reserve4 := defaultReservedIPv4()
	if custom := parseCIDRList(cfg.ReserveIPv4); len(custom) > 0 {
		reserve4 = custom
	}
	reserve6 := defaultReservedIPv6()
	if custom := parseCIDRList(cfg.ReserveIPv6); len(custom) > 0 {
		reserve6 = custom
	}

	routeOut := string(outputIgnore("ip", "route"))
	localIPv4 := parseLocalIPv4(routeOut, true)
	if localIPv4 == "" {
		localIPv4 = parseLocalIPv4(routeOut, false)
	}

	return HostVars{
		HostIPv4:    strings.Join(hosts.IPv4, " "),
		HostIPv6:    strings.Join(hosts.IPv6, " "),
		LocalIPv4:   localIPv4,
		ReserveIPv4: strings.Join(reserve4, " "),
		ReserveIPv6: strings.Join(reserve6, " "),
	}, nil
}

func applyHostVarEnvOverrides(cfg *Config, env map[string]string) {
	read := func(key string) string {
		if env != nil {
			if v, ok := env[key]; ok {
				return strings.TrimSpace(v)
			}
		}
		return strings.TrimSpace(os.Getenv(key))
	}
	override := func(key string, dst *string) {
		if v := read(key); v != "" {
			*dst = v
		}
	}

	override("ipv6_redir", &cfg.IPv6Redir)
	override("ts_service", &cfg.TSService)
	override("wg_service", &cfg.WGService)
	override("replace_default_host_ipv4", &cfg.ReplaceHostV4)
	override("cust_host_ipv4", &cfg.CustHostIPv4)
	override("reserve_ipv4", &cfg.ReserveIPv4)
	override("reserve_ipv6", &cfg.ReserveIPv6)
	override("wg_ipv4", &cfg.WGIPv4)
	override("wg_ipv6", &cfg.WGIPv6)
}

func parseLocalIPv4(routeOut string, filtered bool) string {
	lines := strings.Split(routeOut, "\n")
	seen := map[string]struct{}{}
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if filtered {
			if strings.Contains(line, "utun") || strings.Contains(line, "iot") || strings.Contains(line, "docker") || strings.Contains(line, "linkdown") {
				continue
			}
		}
		match := localSrcIPv4Pattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		ip := match[1]
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	return strings.Join(out, " ")
}
