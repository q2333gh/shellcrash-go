package coreconfig

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

type MenuOptions struct {
	CrashDir string
	TmpDir   string
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
}

type Provider struct {
	Name      string
	Link      string
	LinkURI   string
	Interval  string
	Interval2 string
	UA        string
	ExcludeW  string
	IncludeW  string
}

func RunMenu(opts MenuOptions) error {
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

	s := menuState{
		crashDir: crashDir,
		tmpDir:   tmpDir,
		in:       bufio.NewReader(in),
		out:      out,
		errOut:   errOut,
	}
	return s.mainLoop()
}

type menuState struct {
	crashDir string
	tmpDir   string
	in       *bufio.Reader
	out      io.Writer
	errOut   io.Writer
}

func (s *menuState) mainLoop() error {
	for {
		providers, err := loadProviders(s.crashDir)
		if err != nil {
			return err
		}
		fmt.Fprintln(s.out, "配置文件管理")
		for i, p := range providers {
			link := p.Link
			if link == "" {
				link = p.LinkURI
			}
			if len(link) > 48 {
				link = link[:48] + "..."
			}
			fmt.Fprintf(s.out, "%d) %s\t%s\n", i+1, p.Name, link)
		}
		fmt.Fprintln(s.out, "a) 添加提供者")
		fmt.Fprintln(s.out, "b) 在线生成配置文件")
		fmt.Fprintln(s.out, "d) 清空提供者列表")
		fmt.Fprintln(s.out, "0) 返回")
		choice := s.prompt("请输入对应字母或数字> ")
		switch choice {
		case "", "0":
			return nil
		case "a":
			if err := s.editProvider(Provider{}); err != nil {
				fmt.Fprintf(s.errOut, "添加提供者失败: %v\n", err)
			}
		case "b":
			if _, err := Run(Options{CrashDir: s.crashDir, TmpDir: s.tmpDir}); err != nil {
				fmt.Fprintf(s.errOut, "生成配置失败: %v\n", err)
			} else {
				fmt.Fprintln(s.out, "生成完成")
			}
		case "d":
			if s.prompt("确认清空所有提供者？(1是/0否)> ") == "1" {
				if err := os.Remove(filepath.Join(s.crashDir, "configs", "providers.cfg")); err != nil && !os.IsNotExist(err) {
					return err
				}
				if err := os.Remove(filepath.Join(s.crashDir, "configs", "providers_uri.cfg")); err != nil && !os.IsNotExist(err) {
					return err
				}
				fmt.Fprintln(s.out, "已清空")
			}
		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(providers) {
				fmt.Fprintln(s.out, "输入错误")
				continue
			}
			if err := s.editProvider(providers[idx-1]); err != nil {
				fmt.Fprintf(s.errOut, "管理提供者失败: %v\n", err)
			}
		}
	}
}

func (s *menuState) editProvider(p Provider) error {
	if p.Interval == "" {
		p.Interval = "3"
	}
	if p.Interval2 == "" {
		p.Interval2 = "12"
	}
	if p.UA == "" {
		p.UA = "clash.meta"
	}
	lastName := p.Name
	for {
		fmt.Fprintf(s.out, "当前提供者: name=%s link=%s%s\n", p.Name, p.Link, p.LinkURI)
		fmt.Fprintln(s.out, "1) 设置名称")
		fmt.Fprintln(s.out, "2) 设置链接或路径")
		if p.Link != "" {
			fmt.Fprintln(s.out, "3) 设置本地生成参数")
		}
		fmt.Fprintln(s.out, "a) 保存提供者")
		if p.Name != "" {
			fmt.Fprintln(s.out, "c) 在线生成仅此提供者配置")
			fmt.Fprintln(s.out, "d) 删除此提供者")
			if p.Link != "" {
				fmt.Fprintln(s.out, "e) 直接使用此配置")
			}
		}
		fmt.Fprintln(s.out, "0) 返回")
		choice := s.prompt("请输入对应字母或数字> ")
		switch choice {
		case "", "0":
			return nil
		case "1":
			name := strings.TrimSpace(s.prompt("请输入名称> "))
			if name == "" {
				fmt.Fprintln(s.out, "名称不能为空")
				continue
			}
			if isNumeric(name) {
				fmt.Fprintln(s.out, "名称不能为纯数字")
				continue
			}
			if len(name) > 12 {
				name = name[:12]
			}
			p.Name = name
		case "2":
			text := strings.TrimSpace(strings.ReplaceAll(s.prompt("请输入订阅链接/分享链接/本地文件(例: ./providers/a.yaml)> "), " ", ""))
			if text == "" {
				fmt.Fprintln(s.out, "链接不能为空")
				continue
			}
			switch {
			case strings.HasPrefix(text, "http"):
				p.Link = text
				p.LinkURI = ""
			case strings.HasPrefix(text, "./providers/"):
				if _, err := os.Stat(filepath.Join(s.crashDir, strings.TrimPrefix(text, "./"))); err != nil {
					fmt.Fprintln(s.out, "本地文件不存在")
					continue
				}
				p.Link = text
				p.LinkURI = ""
			default:
				if !isURIExpression(text) {
					fmt.Fprintln(s.out, "链接格式不支持")
					continue
				}
				p.Link = ""
				p.LinkURI = stripComment(text)
			}
		case "3":
			if p.Link == "" {
				fmt.Fprintln(s.out, "仅订阅/本地文件提供者可设置")
				continue
			}
			s.editProviderOptions(&p)
		case "a":
			if err := saveProvider(s.crashDir, lastName, p); err != nil {
				return err
			}
			lastName = p.Name
			fmt.Fprintln(s.out, "保存成功")
		case "c":
			if err := saveProvider(s.crashDir, lastName, p); err != nil {
				return err
			}
			if err := s.setProviderAsSource(p, false); err != nil {
				return err
			}
			if _, err := Run(Options{CrashDir: s.crashDir, TmpDir: s.tmpDir}); err != nil {
				return err
			}
			fmt.Fprintln(s.out, "生成完成")
		case "d":
			if strings.TrimSpace(p.Name) == "" {
				continue
			}
			if err := deleteProviderByName(s.crashDir, p.Name); err != nil {
				return err
			}
			fmt.Fprintln(s.out, "删除成功")
			return nil
		case "e":
			if err := saveProvider(s.crashDir, lastName, p); err != nil {
				return err
			}
			if err := s.setProviderAsSource(p, true); err != nil {
				return err
			}
			if _, err := Run(Options{CrashDir: s.crashDir, TmpDir: s.tmpDir}); err != nil {
				return err
			}
			fmt.Fprintln(s.out, "已应用")
		default:
			fmt.Fprintln(s.out, "输入错误")
		}
	}
}

func (s *menuState) editProviderOptions(p *Provider) {
	for {
		fmt.Fprintf(s.out, "本地生成参数: interval=%s interval2=%s ua=%s exclude=%s include=%s\n", p.Interval, p.Interval2, p.UA, p.ExcludeW, p.IncludeW)
		fmt.Fprintln(s.out, "1) 健康检查间隔(分钟)")
		fmt.Fprintln(s.out, "2) 自动更新间隔(小时)")
		fmt.Fprintln(s.out, "3) 虚拟浏览器UA")
		fmt.Fprintln(s.out, "4) 排除节点正则")
		fmt.Fprintln(s.out, "5) 包含节点正则")
		fmt.Fprintln(s.out, "0) 返回")
		choice := s.prompt("请输入对应数字> ")
		switch choice {
		case "", "0":
			return
		case "1":
			v := strings.TrimSpace(s.prompt("请输入分钟数(r重置)> "))
			if v == "r" {
				p.Interval = "3"
			} else if isPositiveInt(v) {
				p.Interval = v
			}
		case "2":
			v := strings.TrimSpace(s.prompt("请输入小时数(r重置)> "))
			if v == "r" {
				p.Interval2 = "12"
			} else if isPositiveInt(v) {
				p.Interval2 = v
			}
		case "3":
			v := strings.TrimSpace(s.prompt("请输入UA(r重置)> "))
			if v == "r" {
				p.UA = "clash.meta"
			} else if v != "" {
				p.UA = v
			}
		case "4":
			v := strings.ReplaceAll(strings.TrimSpace(s.prompt("请输入排除规则(c清空)> ")), " ", "")
			if v == "c" {
				p.ExcludeW = ""
			} else {
				p.ExcludeW = v
			}
		case "5":
			v := strings.ReplaceAll(strings.TrimSpace(s.prompt("请输入包含规则(c清空)> ")), " ", "")
			if v == "c" {
				p.IncludeW = ""
			} else {
				p.IncludeW = v
			}
		}
	}
}

func (s *menuState) setProviderAsSource(p Provider, direct bool) error {
	cfgPath := filepath.Join(s.crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	if direct && p.Link != "" {
		if strings.HasPrefix(p.Link, "./providers/") {
			target := filepath.Join(s.crashDir, strings.TrimPrefix(p.Link, "./"))
			if _, err := os.Stat(target); err != nil {
				return err
			}
			configPath := clashOrSingboxConfigPath(s.crashDir, cfg["crashcore"])
			if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
				return err
			}
			if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := os.Symlink(target, configPath); err != nil {
				return err
			}
			return nil
		}
		if strings.HasPrefix(p.Link, "http") {
			cfg["Https"] = "'" + p.Link + "'"
			delete(cfg, "Url")
			return writeCfg(cfgPath, cfg)
		}
	}
	if p.Link != "" {
		cfg["Url"] = "'" + p.Link + "'"
		delete(cfg, "Https")
	} else if p.LinkURI != "" {
		url := p.LinkURI
		if p.Name != "vmess" {
			url += "#" + p.Name
		}
		cfg["Url"] = "'" + url + "'"
		delete(cfg, "Https")
	}
	return writeCfg(cfgPath, cfg)
}

func clashOrSingboxConfigPath(crashDir string, crashCoreRaw string) string {
	if strings.Contains(stripQuotes(crashCoreRaw), "singbox") {
		return filepath.Join(crashDir, "jsons", "config.json")
	}
	return filepath.Join(crashDir, "yamls", "config.yaml")
}

func loadProviders(crashDir string) ([]Provider, error) {
	cfgPath := filepath.Join(crashDir, "configs", "providers.cfg")
	uriPath := filepath.Join(crashDir, "configs", "providers_uri.cfg")
	providers := make([]Provider, 0, 16)
	if rows, err := readLinesIfExists(cfgPath); err != nil {
		return nil, err
	} else {
		for _, row := range rows {
			fields := strings.Fields(strings.TrimSpace(row))
			if len(fields) < 2 {
				continue
			}
			p := Provider{
				Name: fields[0],
				Link: fields[1],
			}
			if len(fields) > 2 {
				p.Interval = fields[2]
			}
			if len(fields) > 3 {
				p.Interval2 = fields[3]
			}
			if len(fields) > 4 {
				p.UA = fields[4]
			}
			if len(fields) > 5 {
				p.ExcludeW = strings.TrimPrefix(fields[5], "#")
			}
			if len(fields) > 6 {
				p.IncludeW = strings.TrimPrefix(fields[6], "#")
			}
			if p.Interval == "" {
				p.Interval = "3"
			}
			if p.Interval2 == "" {
				p.Interval2 = "12"
			}
			if p.UA == "" {
				p.UA = "clash.meta"
			}
			providers = append(providers, p)
		}
	}
	if rows, err := readLinesIfExists(uriPath); err != nil {
		return nil, err
	} else {
		for _, row := range rows {
			fields := strings.Fields(strings.TrimSpace(row))
			if len(fields) < 2 {
				continue
			}
			providers = append(providers, Provider{Name: fields[0], LinkURI: fields[1]})
		}
	}
	sort.SliceStable(providers, func(i, j int) bool { return providers[i].Name < providers[j].Name })
	return providers, nil
}

func saveProvider(crashDir, lastName string, p Provider) error {
	if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Link+p.LinkURI) == "" {
		return fmt.Errorf("provider name/link required")
	}
	if p.Interval == "" {
		p.Interval = "3"
	}
	if p.Interval2 == "" {
		p.Interval2 = "12"
	}
	if p.UA == "" {
		p.UA = "clash.meta"
	}
	providers, err := loadProviders(crashDir)
	if err != nil {
		return err
	}
	out := make([]Provider, 0, len(providers)+1)
	for _, item := range providers {
		if item.Name == p.Name || (lastName != "" && item.Name == lastName) {
			continue
		}
		out = append(out, item)
	}
	out = append(out, p)
	return writeProviders(crashDir, out)
}

func deleteProviderByName(crashDir, name string) error {
	providers, err := loadProviders(crashDir)
	if err != nil {
		return err
	}
	out := make([]Provider, 0, len(providers))
	for _, p := range providers {
		if p.Name == name {
			continue
		}
		out = append(out, p)
	}
	return writeProviders(crashDir, out)
}

func writeProviders(crashDir string, providers []Provider) error {
	cfgPath := filepath.Join(crashDir, "configs", "providers.cfg")
	uriPath := filepath.Join(crashDir, "configs", "providers_uri.cfg")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		return err
	}
	var cfgRows []string
	var uriRows []string
	for _, p := range providers {
		if p.Name == "" {
			continue
		}
		if p.Link != "" {
			cfgRows = append(cfgRows, strings.TrimSpace(strings.Join([]string{
				p.Name,
				p.Link,
				emptyDefault(p.Interval, "3"),
				emptyDefault(p.Interval2, "12"),
				emptyDefault(p.UA, "clash.meta"),
				"#" + p.ExcludeW,
				"#" + p.IncludeW,
			}, " ")))
			continue
		}
		if p.LinkURI != "" {
			uriRows = append(uriRows, strings.TrimSpace(p.Name+" "+p.LinkURI))
		}
	}
	if len(cfgRows) == 0 {
		if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if err := os.WriteFile(cfgPath, []byte(strings.Join(cfgRows, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	if len(uriRows) == 0 {
		if err := os.Remove(uriPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if err := os.WriteFile(uriPath, []byte(strings.Join(uriRows, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

func readLinesIfExists(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

func (s *menuState) prompt(label string) string {
	fmt.Fprint(s.out, label)
	line, err := s.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return ""
	}
	return strings.TrimSpace(line)
}

func isURIExpression(v string) bool {
	prefixes := []string{"ss://", "vmess://", "vless://", "trojan://", "tuic://", "anytls://", "shadowtls://", "hysteria://", "hysteria2://"}
	for _, p := range prefixes {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	return false
}

func stripComment(v string) string {
	if i := strings.Index(v, "#"); i >= 0 {
		return v[:i]
	}
	return v
}

func isNumeric(v string) bool {
	if v == "" {
		return false
	}
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isPositiveInt(v string) bool {
	n, err := strconv.Atoi(v)
	return err == nil && n > 0
}

func emptyDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}
