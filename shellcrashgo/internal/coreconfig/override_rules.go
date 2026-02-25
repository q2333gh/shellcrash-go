package coreconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var overrideRulesHTTPDo = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	return client.Do(req)
}

func RunOverrideRulesMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)

	for {
		cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
		cfg, err := readCfg(cfgPath)
		if err != nil {
			return err
		}
		crashCore := stripQuotes(cfg["crashcore"])
		proxiesBypass := stripQuotes(cfg["proxies_bypass"])
		if proxiesBypass == "" {
			proxiesBypass = "OFF"
		}

		fmt.Fprintln(out, "自定义规则")
		fmt.Fprintln(out, "1) 新增自定义规则")
		fmt.Fprintln(out, "2) 移除自定义规则")
		fmt.Fprintln(out, "3) 清空规则列表")
		if !strings.Contains(crashCore, "singbox") {
			fmt.Fprintf(out, "4) 配置节点绕过 [%s]\n", proxiesBypass)
		}
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应数字> ")
		choice, err := readSubLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := runAddOverrideRule(crashDir, cfg, reader, out); err != nil {
				return err
			}
		case "2":
			if err := runDeleteOverrideRule(crashDir, reader, out); err != nil {
				return err
			}
		case "3":
			if err := runClearOverrideRules(crashDir, reader, out); err != nil {
				return err
			}
		case "4":
			if strings.Contains(crashCore, "singbox") {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			next := "ON"
			if strings.EqualFold(proxiesBypass, "ON") {
				next = "OFF"
			} else {
				fmt.Fprintln(out, "启用节点绕过会将节点域名或IP设置为直连，继续？")
				fmt.Fprint(out, "请输入(1=是/0=否)> ")
				confirm, err := readSubLine(reader)
				if err != nil {
					return err
				}
				if confirm != "1" {
					continue
				}
			}
			cfg["proxies_bypass"] = next
			if err := writeCfg(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Fprintln(out, "设置成功")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func runAddOverrideRule(crashDir string, cfg map[string]string, reader *bufio.Reader, out io.Writer) error {
	ruleTypes := []string{
		"DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "IP-CIDR", "SRC-IP-CIDR", "DST-PORT",
		"SRC-PORT", "GEOIP", "GEOSITE", "IP-CIDR6", "DOMAIN", "PROCESS-NAME",
	}
	fmt.Fprintln(out, "请选择规则类型:")
	for i, t := range ruleTypes {
		fmt.Fprintf(out, "%d) %s\n", i+1, t)
	}
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应数字> ")
	choice, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if choice == "" || choice == "0" {
		return nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(ruleTypes) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	ruleType := ruleTypes[idx-1]

	fmt.Fprint(out, "请输入规则语句> ")
	ruleState, err := readSubLine(reader)
	if err != nil {
		return err
	}
	ruleState = strings.TrimSpace(ruleState)
	if ruleState == "" {
		fmt.Fprintln(out, "输入错误")
		return nil
	}

	groups := []string{"DIRECT", "REJECT"}
	dbPort := atoiDefault(stripQuotes(cfg["db_port"]), 9999)
	for _, g := range fetchRuleGroups(dbPort) {
		if g == "" || containsString(groups, g) {
			continue
		}
		groups = append(groups, g)
	}
	fmt.Fprintln(out, "请选择策略组:")
	for i, g := range groups {
		fmt.Fprintf(out, "%d) %s\n", i+1, g)
	}
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应数字> ")
	groupChoice, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if groupChoice == "" || groupChoice == "0" {
		return nil
	}
	groupIdx, err := strconv.Atoi(groupChoice)
	if err != nil || groupIdx < 1 || groupIdx > len(groups) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	ruleGroup := groups[groupIdx-1]

	rule := fmt.Sprintf("- %s,%s,%s", ruleType, ruleState, ruleGroup)
	if ruleType == "IP-CIDR" || ruleType == "SRC-IP-CIDR" || ruleType == "IP-CIDR6" {
		rule += ",no-resolve"
	}
	rulesPath := filepath.Join(crashDir, "yamls", "rules.yaml")
	if err := os.MkdirAll(filepath.Dir(rulesPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(rulesPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(rule + "\n"); err != nil {
		return err
	}
	fmt.Fprintln(out, "添加成功")
	return nil
}

func runDeleteOverrideRule(crashDir string, reader *bufio.Reader, out io.Writer) error {
	rulesPath := filepath.Join(crashDir, "yamls", "rules.yaml")
	lines, err := os.ReadFile(rulesPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	active := filterActiveRuleLines(string(lines))
	if len(active) == 0 {
		fmt.Fprintln(out, "请先添加自定义规则")
		return nil
	}
	for i, line := range active {
		fmt.Fprintf(out, "%d) %s\n", i+1, line)
	}
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应数字> ")
	choice, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if choice == "" || choice == "0" {
		return nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(active) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	active = append(active[:idx-1], active[idx:]...)
	var b strings.Builder
	for _, line := range active {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(rulesPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(rulesPath, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(out, "删除成功")
	return nil
}

func runClearOverrideRules(crashDir string, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprint(out, "确认清空全部自定义规则？(1是/0否)> ")
	confirm, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if confirm != "1" {
		return nil
	}

	rulesPath := filepath.Join(crashDir, "yamls", "rules.yaml")
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "规则列表已清空")
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			if strings.TrimRight(line, "\r") != "" {
				b.WriteString(strings.TrimRight(line, "\r"))
				b.WriteByte('\n')
			}
		}
	}
	if err := os.WriteFile(rulesPath, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(out, "规则列表已清空")
	return nil
}

func filterActiveRuleLines(content string) []string {
	out := make([]string, 0)
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func fetchRuleGroups(dbPort int) []string {
	if dbPort <= 0 {
		return nil
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/proxies", dbPort)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := overrideRulesHTTPDo(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var payload struct {
		Proxies map[string]struct {
			Type string `json:"type"`
		} `json:"proxies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}
	out := make([]string, 0, len(payload.Proxies))
	for name, item := range payload.Proxies {
		switch item.Type {
		case "Selector", "URLTest", "LoadBalance":
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
