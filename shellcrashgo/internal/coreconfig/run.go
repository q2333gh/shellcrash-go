package coreconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	CrashDir string
	TmpDir   string
}

type Result struct {
	Target     string
	Format     string
	CoreConfig string
	Updated    bool
}

func Run(opts Options) (Result, error) {
	if opts.CrashDir == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	if opts.TmpDir == "" {
		opts.TmpDir = "/tmp/ShellCrash"
	}

	cfgPath := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	serversPath := filepath.Join(opts.CrashDir, "configs", "servers.list")
	if _, err := os.Stat(serversPath); errors.Is(err, os.ErrNotExist) {
		serversPath = filepath.Join(opts.CrashDir, "public", "servers.list")
	}

	cfg, err := readCfg(cfgPath)
	if err != nil {
		return Result{}, err
	}
	crashCore := stripQuotes(cfg["crashcore"])
	target, format := "clash", "yaml"
	if strings.Contains(crashCore, "singbox") {
		target, format = "singbox", "json"
	}

	coreConfig := filepath.Join(opts.CrashDir, format+"s", "config."+format)
	coreConfigNew := filepath.Join(opts.TmpDir, target+"_config."+format)
	if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(opts.TmpDir, 0o755); err != nil {
		return Result{}, err
	}

	servers, rules, err := parseServersList(serversPath)
	if err != nil {
		return Result{}, err
	}

	ruleLink := atoiDefault(stripQuotes(cfg["rule_link"]), 1)
	if ruleLink <= 0 || ruleLink > len(rules) {
		ruleLink = 1
	}

	serverLink := atoiDefault(stripQuotes(cfg["server_link"]), 1)
	if serverLink <= 0 || serverLink > len(servers) {
		serverLink = 1
	}

	retry := 0
	for {
		httpsRaw := stripQuotes(cfg["Https"])
		urlType := httpsRaw == ""

		server := servers[serverLink-1]
		rule := rules[ruleLink-1]
		serverURL := server[2]
		serverUA := "ua"
		if len(server) >= 4 && server[3] != "" {
			serverUA = server[3]
		}
		configURL := rule[2]

		ua := buildUserAgent(stripQuotes(cfg["user_agent"]), crashCore, stripQuotes(cfg["core_v"]))
		if urlType {
			subURL := stripQuotes(cfg["Url"])
			if subURL == "" {
				return Result{}, fmt.Errorf("Url and Https are both empty")
			}
			params := url.Values{}
			params.Set("target", target)
			params.Set(serverUA, ua)
			params.Set("insert", "true")
			params.Set("new_name", "true")
			params.Set("scv", "true")
			params.Set("udp", "true")
			params.Set("exclude", stripQuotes(cfg["exclude"]))
			params.Set("include", stripQuotes(cfg["include"]))
			params.Set("url", subURL)
			params.Set("config", configURL)
			httpsRaw = strings.TrimRight(serverURL, "/") + "/sub?" + params.Encode()
		}
		httpsRaw = strings.ReplaceAll(httpsRaw, `\\&`, `&`)

		content, fetchErr := fetch(httpsRaw, ua)
		if fetchErr != nil {
			if !urlType {
				return Result{}, fmt.Errorf("download failed: %w", fetchErr)
			}
			retry++
			if retry >= 3 {
				return Result{}, fmt.Errorf("download failed after %d retries: %w", retry, fetchErr)
			}
			serverLink++
			if serverLink > len(servers) {
				serverLink = 1
			}
			cfg["server_link"] = strconv.Itoa(serverLink)
			if err := writeCfg(cfgPath, cfg); err != nil {
				return Result{}, err
			}
			cfg["Https"] = ""
			continue
		}

		if err := os.WriteFile(coreConfigNew, content, 0o644); err != nil {
			return Result{}, err
		}
		if target == "singbox" {
			content = sanitizeSingboxConfig(content, urlType)
			if err := os.WriteFile(coreConfigNew, content, 0o644); err != nil {
				return Result{}, err
			}
		}
		if err := validateConfig(content, target); err != nil {
			return Result{}, err
		}

		cfg["server_link"] = strconv.Itoa(serverLink)
		cfg["rule_link"] = strconv.Itoa(ruleLink)
		cfg["Https"] = ""
		if ua != "" {
			cfg["user_agent"] = ua
		}
		if err := writeCfg(cfgPath, cfg); err != nil {
			return Result{}, err
		}

		updated, err := replaceIfChanged(coreConfigNew, coreConfig)
		if err != nil {
			return Result{}, err
		}
		return Result{Target: target, Format: format, CoreConfig: coreConfig, Updated: updated}, nil
	}
}

func parseServersList(path string) ([][]string, [][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	servers := make([][]string, 0, 8)
	rules := make([][]string, 0, 8)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		switch {
		case strings.HasPrefix(parts[0], "3") || strings.HasPrefix(parts[0], "4"):
			servers = append(servers, parts)
		case strings.HasPrefix(parts[0], "5"):
			rules = append(rules, parts)
		}
	}
	if len(servers) == 0 || len(rules) == 0 {
		return nil, nil, fmt.Errorf("invalid servers.list")
	}
	return servers, rules, nil
}

func readCfg(path string) (map[string]string, error) {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

func writeCfg(path string, cfg map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		v := cfg[k]
		if strings.Contains(v, " ") && !strings.HasPrefix(v, "'") && !strings.HasPrefix(v, "\"") {
			v = "'" + v + "'"
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func buildUserAgent(ua, crashCore, coreV string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" || ua == "auto" {
		switch {
		case strings.Contains(crashCore, "singbox"):
			if coreV == "" {
				return "sing-box"
			}
			return "sing-box/" + coreV
		case crashCore == "meta":
			return "clash.meta/mihomo"
		default:
			return "clash"
		}
	}
	if ua == "none" {
		return ""
	}
	return ua
}

func fetch(rawURL string, userAgent string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "file" {
		return os.ReadFile(u.Path)
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	cli := &http.Client{Timeout: 20 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func validateConfig(content []byte, target string) error {
	text := string(content)
	if target == "singbox" {
		if !containsAny(text, []string{
			`"socks"`, `"http"`, `"shadowsocks"`, `"shadowsocksr"`, `"vmess"`, `"trojan"`, `"wireguard"`,
			`"hysteria"`, `"hysteria2"`, `"vless"`, `"shadowtls"`, `"tuic"`, `"ssh"`, `"tor"`, `"providers"`,
			`"anytls"`, `"soduku"`,
		}) {
			return fmt.Errorf("singbox config contains no valid nodes/providers")
		}
		if strings.Contains(text, `"sni"`) {
			return fmt.Errorf("unsupported legacy singbox config (<1.12)")
		}
		return nil
	}

	lower := strings.ToLower(text)
	if !containsAny(lower, []string{"proxy-providers:", "server:", "server\":", "server':"}) {
		return fmt.Errorf("clash config contains no valid nodes/providers")
	}
	if strings.Contains(text, "Proxy Group:") {
		return fmt.Errorf("legacy clash config format is unsupported")
	}
	if strings.Contains(text, "cipher: chacha20,") {
		return fmt.Errorf("unsupported chacha20 cipher in config")
	}
	return nil
}

func containsAny(s string, items []string) bool {
	for _, item := range items {
		if strings.Contains(s, item) {
			return true
		}
	}
	return false
}

var (
	singboxDirectURLTestTagPattern = regexp.MustCompile(`\{"type":"urltest","tag":"([^"]*)","outbounds":\["DIRECT"\]\}`)
	singboxDirectURLTestPattern    = regexp.MustCompile(`\{"type":"urltest","tag":"[^"]*","outbounds":\["DIRECT"\]\}`)
	singboxDirectOutboundPattern   = regexp.MustCompile(`\{"type":"[^"]*","tag":"[^"]*","outbounds":\["DIRECT"\],"url":"[^"]*","interval":"[^"]*","tolerance":[^}]*\}`)
	singboxDNSOutPattern           = regexp.MustCompile(`\{[^{}]*"dns-out"[^{}]*\}`)
	singboxCommaPattern            = regexp.MustCompile(`,+`)
)

func sanitizeSingboxConfig(content []byte, urlType bool) []byte {
	text := string(content)
	if strings.Count(text, "\n") < 2 {
		text = strings.Replace(text, `"inbounds":`, `{"inbounds":`, 1)
		text = singboxDNSOutPattern.ReplaceAllString(text, "")
	}
	if !urlType {
		return []byte(cleanSingboxJSONCommas(text))
	}

	tagMatches := singboxDirectURLTestTagPattern.FindAllStringSubmatch(text, -1)
	tags := make([]string, 0, len(tagMatches))
	for _, match := range tagMatches {
		if len(match) > 1 && match[1] != "" {
			tags = append(tags, match[1])
		}
	}
	text = singboxDirectURLTestPattern.ReplaceAllString(text, "")
	text = singboxDirectOutboundPattern.ReplaceAllString(text, "")
	for _, tag := range tags {
		text = strings.ReplaceAll(text, `"`+tag+`"`, "")
	}
	return []byte(cleanSingboxJSONCommas(text))
}

func cleanSingboxJSONCommas(s string) string {
	s = singboxCommaPattern.ReplaceAllString(s, ",")
	s = strings.ReplaceAll(s, "[,", "[")
	s = strings.ReplaceAll(s, ",]", "]")
	return s
}

func replaceIfChanged(src, dst string) (bool, error) {
	newBytes, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	if oldBytes, err := os.ReadFile(dst); err == nil {
		if bytes.Equal(oldBytes, newBytes) {
			_ = os.Remove(src)
			return false, nil
		}
		if err := copyFile(dst, dst+".bak"); err != nil {
			return false, err
		}
	}
	if err := os.Rename(src, dst); err != nil {
		return false, err
	}
	return true, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func stripQuotes(v string) string {
	return strings.Trim(strings.Trim(strings.TrimSpace(v), "'"), "\"")
}

func atoiDefault(v string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}
