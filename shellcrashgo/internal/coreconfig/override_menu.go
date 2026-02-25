package coreconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var groupNamePattern = regexp.MustCompile(`^[^\s,{}\[\]()]+$`)

func RunOverrideMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)

	for {
		cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
		cfg, err := readCfg(cfgPath)
		if err != nil {
			return err
		}
		crashCore := stripQuotes(cfg["crashcore"])
		disOverride := stripQuotes(cfg["disoverride"])

		fmt.Fprintln(out, "配置文件覆写")
		fmt.Fprintln(out, "2) 管理自定义规则")
		if !strings.Contains(crashCore, "singbox") {
			fmt.Fprintln(out, "3) 管理自定义节点")
			fmt.Fprintln(out, "4) 管理自定义策略组")
		}
		fmt.Fprintln(out, "5) 自定义高级功能")
		if disOverride != "1" {
			fmt.Fprintln(out, "9) 禁用配置文件覆写")
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
		case "2":
			if err := RunOverrideRulesMenu(MenuOptions{CrashDir: crashDir, In: reader, Out: out}); err != nil {
				return err
			}
		case "3":
			if strings.Contains(crashCore, "singbox") {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			if err := RunOverrideProxiesMenu(MenuOptions{CrashDir: crashDir, In: reader, Out: out}); err != nil {
				return err
			}
		case "4":
			if strings.Contains(crashCore, "singbox") {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			if err := RunOverrideGroupsMenu(MenuOptions{CrashDir: crashDir, In: reader, Out: out}); err != nil {
				return err
			}
		case "5":
			if strings.Contains(crashCore, "singbox") {
				if err := RunOverrideSingboxAdvanced(MenuOptions{CrashDir: crashDir, In: reader, Out: out}); err != nil {
					return err
				}
			} else {
				if err := RunOverrideClashAdvanced(MenuOptions{CrashDir: crashDir, In: reader, Out: out}); err != nil {
					return err
				}
			}
		case "9":
			if disOverride == "1" {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			fmt.Fprintln(out, "禁用后脚本中大部分覆写功能将不可用，继续？")
			fmt.Fprint(out, "请输入(1=是/0=否)> ")
			confirm, err := readSubLine(reader)
			if err != nil {
				return err
			}
			if confirm != "1" {
				continue
			}
			cfg["disoverride"] = "1"
			if err := writeCfg(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Fprintln(out, "设置成功")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunOverrideGroupsMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	groupsPath := filepath.Join(crashDir, "yamls", "proxy-groups.yaml")

	for {
		fmt.Fprintln(out, "自定义策略组")
		fmt.Fprintln(out, "1) 添加自定义策略组")
		fmt.Fprintln(out, "2) 查看自定义策略组")
		fmt.Fprintln(out, "3) 清空自定义策略组")
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
			if err := runAddOverrideGroup(crashDir, reader, out); err != nil {
				return err
			}
		case "2":
			content, err := os.ReadFile(groupsPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintln(out, "当前无自定义策略组")
					continue
				}
				return err
			}
			fmt.Fprintln(out, string(content))
		case "3":
			fmt.Fprint(out, "确认清空全部自定义策略组？(1是/0否)> ")
			confirm, err := readSubLine(reader)
			if err != nil {
				return err
			}
			if confirm != "1" {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(groupsPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(groupsPath, []byte("#用于添加自定义策略组\n"), 0o644); err != nil {
				return err
			}
			fmt.Fprintln(out, "已清空")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func runAddOverrideGroup(crashDir string, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprint(out, "请输入自定义策略组名称(0返回)> ")
	name, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if name == "" || name == "0" {
		return nil
	}
	if isNumeric(name) || !groupNamePattern.MatchString(name) {
		fmt.Fprintln(out, "策略组名称格式错误")
		return nil
	}

	types := []string{"select", "url-test", "fallback", "load-balance"}
	fmt.Fprintln(out, "请选择策略组类型:")
	for i, t := range types {
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
	if err != nil || idx < 1 || idx > len(types) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	groupType := types[idx-1]

	urlLine := ""
	intervalLine := ""
	if groupType != "select" {
		fmt.Fprint(out, "请输入测速地址(直接回车使用默认)> ")
		url, err := readSubLine(reader)
		if err != nil {
			return err
		}
		if strings.TrimSpace(url) == "" {
			url = "https://www.gstatic.com/generate_204"
		}
		urlLine = fmt.Sprintf("    url: '%s'\n", url)
		intervalLine = "    interval: 300\n"
	}

	allGroups, err := discoverOverrideGroupNames(crashDir)
	if err != nil {
		return err
	}
	if len(allGroups) > 0 {
		fmt.Fprintln(out, "请选择附加到的策略组(空格分隔多个，0跳过):")
		for i, g := range allGroups {
			fmt.Fprintf(out, "%d) %s\n", i+1, g)
		}
		fmt.Fprint(out, "请输入对应数字> ")
		groupChoice, err := readSubLine(reader)
		if err != nil {
			return err
		}
		if strings.TrimSpace(groupChoice) != "" && strings.TrimSpace(groupChoice) != "0" {
			for _, token := range strings.Fields(groupChoice) {
				n, convErr := strconv.Atoi(token)
				if convErr != nil || n < 1 || n > len(allGroups) {
					fmt.Fprintln(out, "输入错误")
					return nil
				}
				name += "#" + allGroups[n-1]
			}
		}
	}

	groupsPath := filepath.Join(crashDir, "yamls", "proxy-groups.yaml")
	if err := os.MkdirAll(filepath.Dir(groupsPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(groupsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	var b strings.Builder
	b.WriteString("  - name: ")
	b.WriteString(name)
	b.WriteByte('\n')
	b.WriteString("    type: ")
	b.WriteString(groupType)
	b.WriteByte('\n')
	b.WriteString(urlLine)
	b.WriteString(intervalLine)
	b.WriteString("    proxies:\n")
	b.WriteString("     - DIRECT\n")
	if _, err := f.WriteString(b.String()); err != nil {
		return err
	}
	fmt.Fprintln(out, "添加成功")
	return nil
}

func RunOverrideProxiesMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	proxiesPath := filepath.Join(crashDir, "yamls", "proxies.yaml")
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")

	for {
		cfg, err := readCfg(cfgPath)
		if err != nil {
			return err
		}
		proxiesBypass := stripQuotes(cfg["proxies_bypass"])
		if proxiesBypass == "" {
			proxiesBypass = "OFF"
		}
		fmt.Fprintln(out, "自定义节点")
		fmt.Fprintln(out, "1) 添加自定义节点")
		fmt.Fprintln(out, "2) 管理自定义节点")
		fmt.Fprintln(out, "3) 清空自定义节点")
		fmt.Fprintf(out, "4) 配置节点绕过 [%s]\n", proxiesBypass)
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
			if err := runAddOverrideProxy(crashDir, proxiesPath, reader, out); err != nil {
				return err
			}
		case "2":
			if err := runDeleteOverrideProxy(proxiesPath, reader, out); err != nil {
				return err
			}
		case "3":
			if err := runClearOverrideProxy(proxiesPath, reader, out); err != nil {
				return err
			}
		case "4":
			next := "ON"
			if strings.EqualFold(proxiesBypass, "ON") {
				next = "OFF"
			} else {
				fmt.Fprint(out, "确认启用节点绕过？(1是/0否)> ")
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

func runAddOverrideProxy(crashDir, proxiesPath string, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprint(out, "请输入自定义节点(需以 name: 开头，0返回)> ")
	proxyRaw, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if proxyRaw == "" || proxyRaw == "0" {
		return nil
	}
	if strings.Contains(proxyRaw, "#") {
		fmt.Fprintln(out, "节点内容禁止包含 #")
		return nil
	}
	if !strings.HasPrefix(strings.TrimSpace(proxyRaw), "name:") {
		fmt.Fprintln(out, "节点格式错误")
		return nil
	}

	groups, err := discoverOverrideGroupNames(crashDir)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		fmt.Fprintln(out, "未发现可用策略组")
		return nil
	}
	fmt.Fprintln(out, "请选择策略组(空格分隔多个，0返回):")
	for i, g := range groups {
		fmt.Fprintf(out, "%d) %s\n", i+1, g)
	}
	fmt.Fprint(out, "请输入对应数字> ")
	choice, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if choice == "" || choice == "0" {
		return nil
	}

	suffix := ""
	for _, token := range strings.Fields(choice) {
		idx, convErr := strconv.Atoi(token)
		if convErr != nil || idx < 1 || idx > len(groups) {
			fmt.Fprintln(out, "输入错误")
			return nil
		}
		suffix += "#" + groups[idx-1]
	}
	if suffix == "" {
		fmt.Fprintln(out, "输入错误")
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(proxiesPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(proxiesPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("- {" + proxyRaw + "}" + suffix + "\n"); err != nil {
		return err
	}
	fmt.Fprintln(out, "添加成功")
	return nil
}

func runDeleteOverrideProxy(proxiesPath string, reader *bufio.Reader, out io.Writer) error {
	data, err := os.ReadFile(proxiesPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "请先添加自定义节点")
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	indices := make([]int, 0)
	active := make([]string, 0)
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		indices = append(indices, i)
		active = append(active, line)
	}
	if len(active) == 0 {
		fmt.Fprintln(out, "请先添加自定义节点")
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
	if err != nil || idx < 1 || idx > len(indices) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	lines[indices[idx-1]] = ""
	outLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		outLines = append(outLines, line)
	}
	content := strings.Join(outLines, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(proxiesPath, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(out, "删除成功")
	return nil
}

func runClearOverrideProxy(proxiesPath string, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprint(out, "确认清空全部自定义节点？(1是/0否)> ")
	confirm, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if confirm != "1" {
		return nil
	}
	data, err := os.ReadFile(proxiesPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "已清空")
			return nil
		}
		return err
	}
	var kept []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			if strings.TrimSpace(raw) != "" {
				kept = append(kept, strings.TrimRight(raw, "\r"))
			}
		}
	}
	content := strings.Join(kept, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(proxiesPath, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(out, "已清空")
	return nil
}

func RunOverrideClashAdvanced(opts MenuOptions) error {
	crashDir, _, _, out, _ := normalizeMenuOptions(opts)
	yamlDir := filepath.Join(crashDir, "yamls")
	if err := os.MkdirAll(yamlDir, 0o755); err != nil {
		return err
	}
	userPath := filepath.Join(yamlDir, "user.yaml")
	othersPath := filepath.Join(yamlDir, "others.yaml")
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		content := "#用于编写自定义设定(可参考https://lancellc.gitbook.io/clash/clash-config-file/general 或 https://docs.metacubex.one/function/general)\n" +
			"#端口之类请在脚本中修改，否则不会加载\n" +
			"#port: 7890\n"
		if err := os.WriteFile(userPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(othersPath); os.IsNotExist(err) {
		content := "#用于编写自定义的锚点、入站、proxy-providers、sub-rules、rule-set、script等功能\n" +
			"#可参考 https://github.com/MetaCubeX/Clash.Meta/blob/Meta/docs/config.yaml 或 https://lancellc.gitbook.io/clash/clash-config-file/an-example-configuration-file\n" +
			"#此处内容会被添加在配置文件的“proxy-group：”模块的末尾与“rules：”模块之前的位置\n" +
			"#例如：\n" +
			"#proxy-providers:\n" +
			"#rule-providers:\n" +
			"#sub-rules:\n" +
			"#tunnels:\n" +
			"#script:\n" +
			"#listeners:\n"
		if err := os.WriteFile(othersPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "已创建或确认文件: %s\n", userPath)
	fmt.Fprintf(out, "已创建或确认文件: %s\n", othersPath)
	return nil
}

func RunOverrideSingboxAdvanced(opts MenuOptions) error {
	_, _, _, out, _ := normalizeMenuOptions(opts)
	fmt.Fprintln(out, "支持覆盖脚本设置模块: log dns ntp certificate experimental")
	fmt.Fprintln(out, "支持与内置功能合并模块: endpoints inbounds outbounds providers route services")
	fmt.Fprintln(out, "将对应 json 文件放入 jsons/ 目录后，启动时会自动加载")
	fmt.Fprintln(out, "参考文档: https://juewuy.github.io/nWTjEpkSK")
	return nil
}

func discoverOverrideGroupNames(crashDir string) ([]string, error) {
	paths := []string{
		filepath.Join(crashDir, "yamls", "proxy-groups.yaml"),
		filepath.Join(crashDir, "yamls", "config.yaml"),
	}
	set := map[string]struct{}{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, raw := range strings.Split(string(data), "\n") {
			line := strings.TrimSpace(raw)
			if !strings.HasPrefix(line, "- name:") {
				continue
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, "- name:"))
			name = strings.SplitN(name, "#", 2)[0]
			if name == "" {
				continue
			}
			set[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}
