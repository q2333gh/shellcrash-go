package minimalflow

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Options struct {
	CrashDir string
	TmpDir   string
}

type Paths struct {
	CfgPath         string
	ProvidersCfg    string
	ProvidersURICfg string
	ServersList     string
	CoreConfig      string
	CoreConfigNew   string
}

func Run(opts Options) error {
	if opts.CrashDir == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	if opts.TmpDir == "" {
		opts.TmpDir = "/tmp/ShellCrash"
	}

	p := Paths{
		CfgPath:         filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg"),
		ProvidersCfg:    filepath.Join(opts.CrashDir, "configs", "providers.cfg"),
		ProvidersURICfg: filepath.Join(opts.CrashDir, "configs", "providers_uri.cfg"),
		ServersList:     filepath.Join(opts.CrashDir, "configs", "servers.list"),
		CoreConfig:      filepath.Join(opts.CrashDir, "yamls", "config.yaml"),
		CoreConfigNew:   filepath.Join(opts.TmpDir, "clash_config.yaml"),
	}

	if err := os.MkdirAll(filepath.Join(opts.CrashDir, "configs"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(opts.CrashDir, "yamls"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(opts.TmpDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(p.CfgPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(p.CfgPath, nil, 0o644); err != nil {
			return err
		}
	}

	cfg, err := readCfg(p.CfgPath)
	if err != nil {
		return err
	}

	cfg["crashcore"] = "meta"

	providerURLs, err := parseProvidersCfg(p.ProvidersCfg)
	if err != nil {
		return err
	}
	uriURLs, err := parseProvidersURICfg(p.ProvidersURICfg)
	if err != nil {
		return err
	}

	urlValue := strings.Trim(strings.Join(append(providerURLs, uriURLs...), "|"), "|")
	if urlValue == "" {
		return errors.New("no providers found")
	}

	cfg["Url"] = "'" + urlValue + "'"
	cfg["Https"] = ""

	serverLink, _ := strconv.Atoi(stripQuotes(cfg["server_link"]))
	if serverLink <= 0 {
		serverLink = 1
	}
	ruleLink, _ := strconv.Atoi(stripQuotes(cfg["rule_link"]))
	if ruleLink <= 0 {
		ruleLink = 1
	}

	exclude := stripQuotes(cfg["exclude"])
	include := stripQuotes(cfg["include"])
	userAgent := stripQuotes(cfg["user_agent"])
	if userAgent == "" || userAgent == "auto" {
		userAgent = "clash.meta/mihomo"
	}
	if userAgent == "none" {
		userAgent = ""
	}

	serverRows, ruleRows, err := parseServersList(p.ServersList)
	if err != nil {
		return err
	}
	if serverLink > len(serverRows) || ruleLink > len(ruleRows) {
		return fmt.Errorf("invalid server/rule selection: server_link=%d, rule_link=%d", serverLink, ruleLink)
	}

	serverRow := serverRows[serverLink-1]
	ruleRow := ruleRows[ruleLink-1]

	server := serverRow[2]
	serverUA := "ua"
	if len(serverRow) > 3 && serverRow[3] != "" {
		serverUA = serverRow[3]
	}
	configTpl := ruleRow[2]

	httpsURL := fmt.Sprintf(
		"%s/sub?target=clash&%s=%s&insert=true&new_name=true&scv=true&udp=true&exclude=%s&include=%s&url=%s&config=%s",
		server,
		serverUA,
		escape(userAgent),
		escape(exclude),
		escape(include),
		escape(urlValue),
		escape(configTpl),
	)

	cfg["server_link"] = strconv.Itoa(serverLink)
	cfg["rule_link"] = strconv.Itoa(ruleLink)
	cfg["user_agent"] = userAgent
	cfg["Https"] = "'" + httpsURL + "'"

	if err := writeCfg(p.CfgPath, cfg); err != nil {
		return err
	}

	content, err := fetch(httpsURL)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.CoreConfigNew, content, 0o644); err != nil {
		return err
	}

	low := strings.ToLower(string(content))
	if !strings.Contains(low, "proxies:") && !strings.Contains(low, "proxy-providers:") && !strings.Contains(low, "server:") {
		return errors.New("invalid yaml config")
	}
	if !strings.Contains(low, "hysteria2") && !strings.Contains(low, "type: hysteria") {
		return errors.New("hy2 not found")
	}

	if old, err := os.ReadFile(p.CoreConfig); err == nil {
		if bytes.Equal(old, content) {
			_ = os.Remove(p.CoreConfigNew)
			return nil
		}
		if err := copyFile(p.CoreConfig, p.CoreConfig+".bak"); err != nil {
			return err
		}
	}

	return os.Rename(p.CoreConfigNew, p.CoreConfig)
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
	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(cfg[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func parseProvidersCfg(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && !strings.HasPrefix(parts[1], "./providers/") {
			out = append(out, parts[1])
		}
	}
	return out, nil
}

func parseProvidersURICfg(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		parts := strings.Fields(s)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		link := parts[1]
		if name == "vmess" {
			out = append(out, link)
		} else {
			out = append(out, link+"#"+name)
		}
	}
	return out, nil
}

func parseServersList(path string) ([][]string, [][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var servers [][]string
	var rules [][]string
	for _, line := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		parts := strings.Fields(s)
		if len(parts) < 3 {
			continue
		}
		if strings.HasPrefix(parts[0], "3") || strings.HasPrefix(parts[0], "4") {
			servers = append(servers, parts)
		}
		if strings.HasPrefix(parts[0], "5") {
			rules = append(rules, parts)
		}
	}
	if len(servers) == 0 || len(rules) == 0 {
		return nil, nil, errors.New("servers.list missing server/rule rows")
	}
	return servers, rules, nil
}

func fetch(rawURL string) ([]byte, error) {
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
	client := &http.Client{Timeout: 20 * 1e9}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func stripQuotes(v string) string {
	return strings.Trim(strings.Trim(strings.TrimSpace(v), "'"), "\"")
}

func escape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
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
