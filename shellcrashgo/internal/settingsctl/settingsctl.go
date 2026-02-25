package settingsctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Options struct {
	CrashDir string
}

type State struct {
	CrashCore      string
	RedirMod       string
	DNSMod         string
	DNSNameServer  string
	DNSFallback    string
	DNSResolver    string
	SkipCert       string
	Sniffer        string
	IPv6Redir      string
	IPv6DNS        string
	DNSProtect     string
	HostsOpt       string
	ECSSubnet      string
	MixPort        int
	RedirPort      int
	DNSPort        int
	DBPort         int
	DNSRedirPort   int
	Authentication string
	Secret         string
	Host           string
	Table          int
}

func RunMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "功能设置")
		fmt.Fprintf(out, "1) 路由模式: %s\n", st.RedirMod)
		fmt.Fprintf(out, "2) DNS模式: %s\n", st.DNSMod)
		fmt.Fprintf(out, "3) 跳过本地证书验证: %s\n", st.SkipCert)
		fmt.Fprintf(out, "4) 域名嗅探: %s\n", st.Sniffer)
		fmt.Fprintln(out, "5) 高级端口设置")
		fmt.Fprintf(out, "6) IPv6设置: Redir=%s DNS=%s\n", st.IPv6Redir, st.IPv6DNS)
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := runRedirMenu(opts, reader, out); err != nil {
				return err
			}
		case "2":
			if err := runDNSMenu(opts, reader, out); err != nil {
				return err
			}
		case "3":
			next := "OFF"
			if st.SkipCert == "OFF" {
				next = "ON"
			}
			if err := setCfgValue(opts, "skip_cert", next); err != nil {
				return err
			}
			fmt.Fprintf(out, "skip_cert=%s\n", next)
		case "4":
			next := "OFF"
			if st.Sniffer == "OFF" {
				next = "ON"
			}
			if err := setCfgValue(opts, "sniffer", next); err != nil {
				return err
			}
			fmt.Fprintf(out, "sniffer=%s\n", next)
		case "5":
			if err := runAdvancedPortMenuWithReader(opts, reader, out); err != nil {
				return err
			}
		case "6":
			if err := runIPv6MenuWithReader(opts, reader, out); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunAdvancedPortMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	return runAdvancedPortMenuWithReader(opts, reader, out)
}

func runAdvancedPortMenuWithReader(opts Options, reader *bufio.Reader, out io.Writer) error {
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		auth := "未设置"
		if st.Authentication != "" {
			auth = "******"
		}
		secret := st.Secret
		if secret == "" {
			secret = "未设置"
		}
		fmt.Fprintln(out, "高级端口设置")
		fmt.Fprintf(out, "1) 混合端口: %d\n", st.MixPort)
		fmt.Fprintf(out, "2) 认证信息: %s\n", auth)
		fmt.Fprintf(out, "3) Redir端口: %d,%d\n", st.RedirPort, st.RedirPort+1)
		fmt.Fprintf(out, "4) DNS端口: %d\n", st.DNSPort)
		fmt.Fprintf(out, "5) 面板端口: %d\n", st.DBPort)
		fmt.Fprintf(out, "6) 面板密码: %s\n", secret)
		fmt.Fprintf(out, "8) Host: %s\n", st.Host)
		fmt.Fprintf(out, "9) Table: %d,%d\n", st.Table, st.Table+1)
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := readAndSetPort(opts, reader, out, "mix_port"); err != nil {
				return err
			}
		case "2":
			fmt.Fprint(out, "请输入账号密码(user:pass，输入0清空)> ")
			v, err := readLine(reader)
			if err != nil {
				return err
			}
			if v == "0" {
				if err := setCfgValue(opts, "authentication", ""); err != nil {
					return err
				}
				continue
			}
			if !strings.Contains(v, ":") {
				fmt.Fprintln(out, "输入格式无效")
				continue
			}
			if err := setCfgValue(opts, "authentication", quote(v)); err != nil {
				return err
			}
		case "3":
			if err := readAndSetPort(opts, reader, out, "redir_port"); err != nil {
				return err
			}
		case "4":
			if err := readAndSetPort(opts, reader, out, "dns_port"); err != nil {
				return err
			}
		case "5":
			if err := readAndSetPort(opts, reader, out, "db_port"); err != nil {
				return err
			}
		case "6":
			fmt.Fprint(out, "请输入面板密码(输入0清空)> ")
			v, err := readLine(reader)
			if err != nil {
				return err
			}
			if v == "0" {
				v = ""
			}
			if err := setCfgValue(opts, "secret", v); err != nil {
				return err
			}
		case "8":
			fmt.Fprint(out, "请输入Host(输入0清空)> ")
			v, err := readLine(reader)
			if err != nil {
				return err
			}
			if v == "0" {
				v = ""
			}
			if err := setCfgValue(opts, "host", v); err != nil {
				return err
			}
		case "9":
			fmt.Fprint(out, "请输入路由表号(1-65534，输入0恢复100)> ")
			v, err := readLine(reader)
			if err != nil {
				return err
			}
			n := 100
			if v != "0" {
				n, err = strconv.Atoi(v)
				if err != nil || n < 1 || n > 65534 {
					fmt.Fprintln(out, "输入错误")
					continue
				}
			}
			if err := setCfgValue(opts, "table", strconv.Itoa(n)); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunIPv6Menu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	return runIPv6MenuWithReader(opts, reader, out)
}

func runIPv6MenuWithReader(opts Options, reader *bufio.Reader, out io.Writer) error {
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "IPv6设置")
		fmt.Fprintf(out, "1) IPv6转发: %s\n", st.IPv6Redir)
		fmt.Fprintf(out, "2) IPv6 DNS: %s\n", st.IPv6DNS)
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			next := "OFF"
			if st.IPv6Redir == "OFF" {
				next = "ON"
			}
			if err := setCfgValue(opts, "ipv6_redir", next); err != nil {
				return err
			}
			if next == "ON" {
				if err := setCfgValue(opts, "ipv6_support", "ON"); err != nil {
					return err
				}
			}
		case "2":
			next := "OFF"
			if st.IPv6DNS == "OFF" {
				next = "ON"
			}
			if err := setCfgValue(opts, "ipv6_dns", next); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunDNSMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	return runDNSMenu(opts, reader, out)
}

func RunFakeIPFilterMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	return runFakeIPFilterMenu(opts, reader, out)
}

func RunDNSAdvancedMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	return runDNSAdvancedMenu(opts, reader, out)
}

func LoadState(opts Options) (State, error) {
	opts = withDefaults(opts)
	cfgKV, err := parseKVFile(filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg"))
	if err != nil && !os.IsNotExist(err) {
		return State{}, err
	}
	if cfgKV == nil {
		cfgKV = map[string]string{}
	}
	st := State{}
	st.CrashCore = stripQuotes(cfgKV["crashcore"])
	st.RedirMod = defaultText(stripQuotes(cfgKV["redir_mod"]), "Redir")
	st.DNSMod = defaultText(stripQuotes(cfgKV["dns_mod"]), "redir_host")
	st.DNSNameServer = defaultText(stripQuotes(cfgKV["dns_nameserver"]), "127.0.0.1")
	st.DNSFallback = defaultText(stripQuotes(cfgKV["dns_fallback"]), "1.1.1.1, 8.8.8.8")
	st.DNSResolver = defaultText(stripQuotes(cfgKV["dns_resolver"]), "223.5.5.5, 2400:3200::1")
	st.SkipCert = defaultONOFF(cfgKV["skip_cert"], "ON")
	if strings.TrimSpace(cfgKV["sniffer"]) == "" {
		if strings.Contains(st.CrashCore, "singbox") {
			st.Sniffer = "ON"
		} else {
			st.Sniffer = "OFF"
		}
	} else {
		st.Sniffer = defaultONOFF(cfgKV["sniffer"], "OFF")
	}
	st.IPv6Redir = defaultONOFF(cfgKV["ipv6_redir"], "OFF")
	st.IPv6DNS = defaultONOFF(cfgKV["ipv6_dns"], "ON")
	st.DNSProtect = defaultONOFF(cfgKV["dns_protect"], "ON")
	st.HostsOpt = defaultONOFF(cfgKV["hosts_opt"], "ON")
	st.ECSSubnet = defaultONOFF(cfgKV["ecs_subnet"], "OFF")
	st.MixPort = defaultInt(cfgKV["mix_port"], 7890)
	st.RedirPort = defaultInt(cfgKV["redir_port"], 7892)
	st.DNSPort = defaultInt(cfgKV["dns_port"], 1053)
	st.DBPort = defaultInt(cfgKV["db_port"], 9999)
	st.DNSRedirPort = defaultInt(cfgKV["dns_redir_port"], st.DNSPort)
	st.Authentication = stripQuotes(cfgKV["authentication"])
	st.Secret = stripQuotes(cfgKV["secret"])
	st.Host = stripQuotes(cfgKV["host"])
	st.Table = defaultInt(cfgKV["table"], 100)
	return st, nil
}

func withDefaults(opts Options) Options {
	if strings.TrimSpace(opts.CrashDir) == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	return opts
}

func runRedirMenu(opts Options, reader *bufio.Reader, out io.Writer) error {
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "当前路由模式: %s\n", st.RedirMod)
		fmt.Fprintln(out, "1) Redir")
		fmt.Fprintln(out, "2) Mix")
		fmt.Fprintln(out, "3) Tproxy")
		fmt.Fprintln(out, "4) Tun")
		fmt.Fprintln(out, "5) 纯净模式")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		if choice == "" || choice == "0" {
			return nil
		}
		var redir string
		firewallArea := ""
		switch choice {
		case "1":
			redir = "Redir"
		case "2":
			redir = "Mix"
		case "3":
			redir = "Tproxy"
		case "4":
			redir = "Tun"
		case "5":
			redir = "纯净模式"
			firewallArea = "4"
		default:
			fmt.Fprintln(out, "输入错误")
			continue
		}
		if err := setCfgValue(opts, "redir_mod", redir); err != nil {
			return err
		}
		if firewallArea != "" {
			if err := setCfgValue(opts, "firewall_area", firewallArea); err != nil {
				return err
			}
		}
	}
}

func runDNSMenu(opts Options, reader *bufio.Reader, out io.Writer) error {
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "当前DNS模式: %s\n", st.DNSMod)
		fmt.Fprintln(out, "1) MIX")
		fmt.Fprintln(out, "2) ROUTE")
		fmt.Fprintln(out, "3) REDIR")
		fmt.Fprintf(out, "4) DNS防泄漏: %s\n", st.DNSProtect)
		fmt.Fprintf(out, "5) Hosts优化: %s\n", st.HostsOpt)
		fmt.Fprintf(out, "6) ECS: %s\n", st.ECSSubnet)
		fmt.Fprintf(out, "7) DNS劫持端口: %d\n", st.DNSRedirPort)
		if st.DNSMod == "mix" {
			fmt.Fprintln(out, "8) Fake-IP过滤")
		}
		fmt.Fprintln(out, "9) DNS详细设置")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := setCfgValue(opts, "dns_mod", "mix"); err != nil {
				return err
			}
		case "2":
			if err := setCfgValue(opts, "dns_mod", "route"); err != nil {
				return err
			}
		case "3":
			if err := setCfgValue(opts, "dns_mod", "redir_host"); err != nil {
				return err
			}
		case "4":
			next := flipONOFF(st.DNSProtect)
			if err := setCfgValue(opts, "dns_protect", next); err != nil {
				return err
			}
		case "5":
			next := flipONOFF(st.HostsOpt)
			if err := setCfgValue(opts, "hosts_opt", next); err != nil {
				return err
			}
		case "6":
			next := flipONOFF(st.ECSSubnet)
			if err := setCfgValue(opts, "ecs_subnet", next); err != nil {
				return err
			}
		case "7":
			if err := readAndSetPort(opts, reader, out, "dns_redir_port"); err != nil {
				return err
			}
		case "8":
			if st.DNSMod != "mix" {
				fmt.Fprintln(out, "当前模式不支持")
				continue
			}
			if err := runFakeIPFilterMenu(opts, reader, out); err != nil {
				return err
			}
		case "9":
			if err := runDNSAdvancedMenu(opts, reader, out); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func runFakeIPFilterMenu(opts Options, reader *bufio.Reader, out io.Writer) error {
	path := filepath.Join(opts.CrashDir, "configs", "fake_ip_filter")
	for {
		entries, err := readTextLines(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Fprintln(out, "Fake-IP过滤列表")
		if len(entries) == 0 {
			fmt.Fprintln(out, "(空)")
		} else {
			for i, item := range entries {
				fmt.Fprintf(out, "%d) %s\n", i+1, item)
			}
		}
		fmt.Fprint(out, "输入序号删除，或输入新条目添加，0返回> ")
		raw, err := readLine(reader)
		if err != nil {
			return err
		}
		if raw == "" || raw == "0" {
			return nil
		}
		if idx, convErr := strconv.Atoi(raw); convErr == nil {
			if idx < 1 || idx > len(entries) {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			entries = append(entries[:idx-1], entries[idx:]...)
			if err := writeTextLines(path, entries); err != nil {
				return err
			}
			continue
		}
		entries = append(entries, strings.TrimSpace(raw))
		if err := writeTextLines(path, entries); err != nil {
			return err
		}
	}
}

func runDNSAdvancedMenu(opts Options, reader *bufio.Reader, out io.Writer) error {
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "DNS详细设置")
		fmt.Fprintf(out, "DIRECT-DNS: %s\n", st.DNSNameServer)
		fmt.Fprintf(out, "PROXY-DNS: %s\n", st.DNSFallback)
		fmt.Fprintf(out, "DEFAULT-DNS: %s\n", st.DNSResolver)
		fmt.Fprintln(out, "1) 修改DIRECT-DNS")
		fmt.Fprintln(out, "2) 修改PROXY-DNS")
		fmt.Fprintln(out, "3) 修改DEFAULT-DNS")
		fmt.Fprintln(out, "4) 自动加密DNS")
		fmt.Fprintln(out, "9) 恢复默认")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := readAndSetDNSValue(opts, reader, out, "dns_nameserver", "127.0.0.1"); err != nil {
				return err
			}
		case "2":
			if err := readAndSetDNSValue(opts, reader, out, "dns_fallback", "1.1.1.1, 8.8.8.8"); err != nil {
				return err
			}
		case "3":
			fmt.Fprint(out, "请输入新DEFAULT-DNS(输入r重置，0返回)> ")
			v, err := readLine(reader)
			if err != nil {
				return err
			}
			switch v {
			case "0":
				continue
			case "r":
				if err := setCfgValue(opts, "dns_resolver", quote("223.5.5.5, 2400:3200::1")); err != nil {
					return err
				}
			default:
				if strings.Contains(v, "://") && strings.Contains(v, "::") {
					fmt.Fprintln(out, "不支持IPv6 DoH地址")
					continue
				}
				if err := setCfgValue(opts, "dns_resolver", quote(normalizeDNSInput(v))); err != nil {
					return err
				}
			}
		case "4":
			if strings.Contains(st.CrashCore, "singbox") || st.CrashCore == "meta" {
				if err := setCfgValue(opts, "dns_nameserver", quote("https://dns.alidns.com/dns-query, https://doh.pub/dns-query")); err != nil {
					return err
				}
				if err := setCfgValue(opts, "dns_fallback", quote("https://cloudflare-dns.com/dns-query, https://dns.google/dns-query, https://doh.opendns.com/dns-query")); err != nil {
					return err
				}
				if err := setCfgValue(opts, "dns_resolver", quote("https://223.5.5.5/dns-query, 2400:3200::1")); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(out, "当前内核不支持自动加密DNS")
			}
		case "9":
			if err := setCfgValue(opts, "dns_nameserver", ""); err != nil {
				return err
			}
			if err := setCfgValue(opts, "dns_fallback", ""); err != nil {
				return err
			}
			if err := setCfgValue(opts, "dns_resolver", ""); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func readAndSetDNSValue(opts Options, reader *bufio.Reader, out io.Writer, key, reset string) error {
	fmt.Fprint(out, "请输入新地址(输入r重置，0返回)> ")
	v, err := readLine(reader)
	if err != nil {
		return err
	}
	switch v {
	case "0":
		return nil
	case "r":
		return setCfgValue(opts, key, quote(reset))
	default:
		return setCfgValue(opts, key, quote(normalizeDNSInput(v)))
	}
}

func normalizeDNSInput(v string) string {
	return strings.ReplaceAll(strings.TrimSpace(v), "|", ", ")
}

func readAndSetPort(opts Options, reader *bufio.Reader, out io.Writer, key string) error {
	fmt.Fprint(out, "请输入端口(1-65535)> ")
	raw, err := readLine(reader)
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 65535 {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	return setCfgValue(opts, key, strconv.Itoa(n))
}

func setCfgValue(opts Options, key, value string) error {
	path := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	kv, err := parseKVFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	if strings.TrimSpace(value) == "" {
		delete(kv, key)
	} else {
		kv[key] = value
	}
	return writeKVFile(path, kv)
}

func parseKVFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	kv := map[string]string{}
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
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return kv, s.Err()
}

func readTextLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, s.Err()
}

func writeTextLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(lines) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
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

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
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

func quote(s string) string {
	s = strings.ReplaceAll(s, "'", "")
	return "'" + s + "'"
}

func defaultText(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultONOFF(v, fallback string) string {
	s := strings.ToUpper(strings.TrimSpace(stripQuotes(v)))
	if s == "ON" || s == "OFF" {
		return s
	}
	return fallback
}

func flipONOFF(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), "ON") {
		return "OFF"
	}
	return "ON"
}

func defaultInt(v string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(stripQuotes(v)))
	if err != nil {
		return fallback
	}
	return n
}
