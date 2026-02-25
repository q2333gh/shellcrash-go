package coreconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var subconverterRun = Run

func RunSubconverterMenu(opts MenuOptions) error {
	crashDir, tmpDir, in, out, errOut := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)

	for {
		cfg, err := readCfg(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
		if err != nil {
			return err
		}
		exclude := stripQuotes(cfg["exclude"])
		include := stripQuotes(cfg["include"])
		userAgent := stripQuotes(cfg["user_agent"])

		fmt.Fprintln(out, "Subconverter 在线订阅转换")
		fmt.Fprintln(out, "1) 生成包含全部节点、订阅的配置文件")
		fmt.Fprintf(out, "2) 设置排除节点正则 [%s]\n", exclude)
		fmt.Fprintf(out, "3) 设置包含节点正则 [%s]\n", include)
		fmt.Fprintln(out, "4) 选择在线规则模版")
		fmt.Fprintln(out, "5) 选择Subconverter服务器")
		fmt.Fprintf(out, "6) 自定义浏览器UA [%s]\n", userAgent)
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
			if err := RunSubconverterGenerate(Options{CrashDir: crashDir, TmpDir: tmpDir}); err != nil {
				return err
			}
			fmt.Fprintln(out, "配置生成成功")
		case "2":
			if err := runSubconverterTextEditMenu(crashDir, reader, out, "exclude", "排除节点关键字", errOut); err != nil {
				return err
			}
		case "3":
			if err := runSubconverterTextEditMenu(crashDir, reader, out, "include", "包含节点关键字", errOut); err != nil {
				return err
			}
		case "4":
			if err := RunSubconverterRuleMenu(MenuOptions{CrashDir: crashDir, In: reader, Out: out, Err: errOut}); err != nil {
				return err
			}
		case "5":
			if err := RunSubconverterServerMenu(MenuOptions{CrashDir: crashDir, In: reader, Out: out, Err: errOut}); err != nil {
				return err
			}
		case "6":
			if err := RunSubconverterUAMenu(MenuOptions{CrashDir: crashDir, In: reader, Out: out, Err: errOut}); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunSubconverterGenerate(opts Options) error {
	if strings.TrimSpace(opts.CrashDir) == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	if strings.TrimSpace(opts.TmpDir) == "" {
		opts.TmpDir = "/tmp/ShellCrash"
	}
	cfgPath := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")

	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	combined, err := buildSubconverterURL(opts.CrashDir)
	if err != nil {
		return err
	}
	cfg["Url"] = quoteValue(combined)
	cfg["Https"] = ""
	if err := writeCfg(cfgPath, cfg); err != nil {
		return err
	}
	_, err = subconverterRun(opts)
	return err
}

func RunSubconverterExcludeMenu(opts MenuOptions) error {
	crashDir, _, in, out, errOut := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	return runSubconverterTextEditMenu(crashDir, reader, out, "exclude", "排除节点关键字", errOut)
}

func RunSubconverterIncludeMenu(opts MenuOptions) error {
	crashDir, _, in, out, errOut := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	return runSubconverterTextEditMenu(crashDir, reader, out, "include", "包含节点关键字", errOut)
}

func RunSubconverterRuleMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	_, rules, err := parseServersList(resolveServersListPath(crashDir))
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	current := atoiDefault(stripQuotes(cfg["rule_link"]), 1)
	if current < 1 || current > len(rules) {
		current = 1
	}
	fmt.Fprintf(out, "当前在线规则模版: %s\n", rules[current-1][1])
	for i, item := range rules {
		name := item[1]
		if len(item) > 3 {
			name += item[3]
		}
		fmt.Fprintf(out, "%d) %s\n", i+1, name)
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
	n, err := strconv.Atoi(choice)
	if err != nil || n < 1 || n > len(rules) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	cfg["rule_link"] = strconv.Itoa(n)
	if err := writeCfg(cfgPath, cfg); err != nil {
		return err
	}
	fmt.Fprintln(out, "设置成功")
	return nil
}

func RunSubconverterServerMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	servers, _, err := parseServersList(resolveServersListPath(crashDir))
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	current := atoiDefault(stripQuotes(cfg["server_link"]), 1)
	if current < 1 || current > len(servers) {
		current = 1
	}
	fmt.Fprintf(out, "当前Subconverter服务器: %s\n", servers[current-1][2])
	for i, item := range servers {
		fmt.Fprintf(out, "%d) %s\t%s\n", i+1, item[2], item[1])
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
	n, err := strconv.Atoi(choice)
	if err != nil || n < 1 || n > len(servers) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	cfg["server_link"] = strconv.Itoa(n)
	if err := writeCfg(cfgPath, cfg); err != nil {
		return err
	}
	fmt.Fprintln(out, "设置成功")
	return nil
}

func RunSubconverterUAMenu(opts MenuOptions) error {
	crashDir, _, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	current := stripQuotes(cfg["user_agent"])
	for {
		fmt.Fprintf(out, "当前UA: %s\n", current)
		fmt.Fprintln(out, "1) 使用自动UA")
		fmt.Fprintln(out, "2) 不使用UA")
		fmt.Fprintln(out, "3) 使用自定义UA")
		fmt.Fprintln(out, "4) 清空UA")
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
			current = "auto"
		case "2":
			current = "none"
		case "3":
			fmt.Fprint(out, "请输入自定义UA(0返回)> ")
			v, err := readSubLine(reader)
			if err != nil {
				return err
			}
			v = strings.TrimSpace(v)
			if v == "" || v == "0" {
				continue
			}
			current = v
		case "4":
			current = ""
		default:
			fmt.Fprintln(out, "输入错误")
			continue
		}
		cfg["user_agent"] = current
		if err := writeCfg(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Fprintln(out, "设置成功")
		return nil
	}
}

func runSubconverterTextEditMenu(crashDir string, reader *bufio.Reader, out io.Writer, key, title string, errOut io.Writer) error {
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	now := stripQuotes(cfg[key])
	fmt.Fprintf(out, "当前%s: %s\n", title, now)
	fmt.Fprintln(out, "请输入关键字，多个关键字可用 | 分隔")
	fmt.Fprintln(out, "输入 d 清空，输入 0 返回")
	fmt.Fprint(out, "请输入> ")
	text, err := readSubLine(reader)
	if err != nil {
		return err
	}
	switch text {
	case "0":
		return nil
	case "d":
		cfg[key] = ""
	default:
		cfg[key] = text
	}
	if err := writeCfg(cfgPath, cfg); err != nil {
		return err
	}
	if errOut != nil {
		_, _ = fmt.Fprintln(errOut, "设置成功")
	}
	return nil
}

func buildSubconverterURL(crashDir string) (string, error) {
	links := make([]string, 0, 32)
	cfgPath := filepath.Join(crashDir, "configs", "providers.cfg")
	if data, err := os.ReadFile(cfgPath); err == nil {
		for _, raw := range strings.Split(string(data), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			link := strings.TrimSpace(fields[1])
			if link == "" || strings.Contains(link, "./providers/") {
				continue
			}
			links = append(links, link)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	uriPath := filepath.Join(crashDir, "configs", "providers_uri.cfg")
	if data, err := os.ReadFile(uriPath); err == nil {
		for _, raw := range strings.Split(string(data), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			entry := fields[1]
			if fields[0] != "vmess" {
				entry = fields[1] + "#" + fields[0]
			}
			links = append(links, entry)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	combined := strings.Trim(mergePipeList(strings.Join(links, "|")), "|")
	if combined == "" {
		return "", fmt.Errorf("providers list is empty")
	}
	return combined, nil
}

func mergePipeList(s string) string {
	for strings.Contains(s, "||") {
		s = strings.ReplaceAll(s, "||", "|")
	}
	return s
}

func resolveServersListPath(crashDir string) string {
	path := filepath.Join(crashDir, "configs", "servers.list")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return filepath.Join(crashDir, "public", "servers.list")
}

func normalizeMenuOptions(opts MenuOptions) (string, string, io.Reader, io.Writer, io.Writer) {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	tmpDir := strings.TrimSpace(opts.TmpDir)
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	in := opts.In
	if in == nil {
		in = os.Stdin
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Err
	if errOut == nil {
		errOut = os.Stderr
	}
	return crashDir, tmpDir, in, out, errOut
}

func readSubLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(strings.TrimRight(line, "\r\n")), nil
}

func quoteValue(v string) string {
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "'") || strings.HasPrefix(v, "\"") {
		return v
	}
	return "'" + v + "'"
}
