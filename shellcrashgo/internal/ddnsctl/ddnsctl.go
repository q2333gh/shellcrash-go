package ddnsctl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Options struct {
	ConfigPath   string
	ServicesIPv4 string
	ServicesIPv6 string
	ServicesList string
	LogDir       string
	UpdaterPath  string
	In           io.Reader
	Out          io.Writer
	Err          io.Writer
}

type Deps struct {
	RunCommand func(name string, args ...string) ([]byte, error)
}

type Service struct {
	Name    string
	Domain  string
	Enabled string
	LastIP  string
}

type AddParams struct {
	ServiceID     string
	ServiceName   string
	Domain        string
	Username      string
	Password      string
	UseIPv6       bool
	CheckInterval int
	ForceInterval int
}

func RunMenu(opts Options, deps Deps) error {
	opts = withDefaults(opts)
	deps = withDeps(deps)
	if _, err := os.Stat(opts.ConfigPath); err != nil {
		return fmt.Errorf("ddns config missing: %s", opts.ConfigPath)
	}

	reader := bufio.NewReader(opts.In)
	for {
		services, err := ListServices(opts, deps)
		if err != nil {
			return err
		}
		fmt.Fprintln(opts.Out, "-----------------------------------------------")
		fmt.Fprintln(opts.Out, "列表  域名  启用  IP地址")
		for i, s := range services {
			fmt.Fprintf(opts.Out, "%d) %s  %s  %s\n", i+1, emptyAsDash(s.Domain), emptyAsDash(s.Enabled), emptyAsDash(s.LastIP))
		}
		fmt.Fprintf(opts.Out, "%d) 添加DDNS服务\n", len(services)+1)
		fmt.Fprintln(opts.Out, "0) 退出")
		fmt.Fprint(opts.Out, "请输入对应序号> ")

		choice, err := readInt(reader)
		if err != nil {
			return err
		}
		switch {
		case choice == 0:
			return nil
		case choice == len(services)+1:
			if err := menuAdd(opts, deps, reader); err != nil {
				return err
			}
		case choice > 0 && choice <= len(services):
			if err := menuServiceAction(opts, deps, reader, services[choice-1].Name); err != nil {
				return err
			}
		default:
			fmt.Fprintln(opts.Err, "输入错误")
		}
	}
}

func ListServices(opts Options, deps Deps) ([]Service, error) {
	opts = withDefaults(opts)
	deps = withDeps(deps)

	names, err := listServiceIDs(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	services := make([]Service, 0, len(names))
	for _, name := range names {
		domain, _ := getUCI(deps, name, "domain")
		enabled, _ := getUCI(deps, name, "enabled")
		lastIP := readLastRegisteredIP(filepath.Join(opts.LogDir, name+".log"))
		services = append(services, Service{
			Name:    name,
			Domain:  strings.TrimSpace(domain),
			Enabled: strings.TrimSpace(enabled),
			LastIP:  strings.TrimSpace(lastIP),
		})
	}
	return services, nil
}

func AddService(opts Options, deps Deps, p AddParams) error {
	opts = withDefaults(opts)
	deps = withDeps(deps)

	if strings.TrimSpace(p.ServiceID) == "" || strings.TrimSpace(p.ServiceName) == "" {
		return fmt.Errorf("service id/name required")
	}
	if strings.TrimSpace(p.Domain) == "" {
		return fmt.Errorf("domain required")
	}
	if p.CheckInterval < 1 || p.CheckInterval > 1440 {
		p.CheckInterval = 10
	}
	if p.ForceInterval < 1 || p.ForceInterval > 240 {
		p.ForceInterval = 24
	}

	useIPv6 := "0"
	if p.UseIPv6 {
		useIPv6 = "1"
	}
	block := fmt.Sprintf(`
config service '%s'
	option enabled '1'
	option force_unit 'hours'
	option lookup_host '%s'
	option service_name '%s'
	option domain '%s'
	option username '%s'
	option use_https '0'
	option use_ipv6 '%s'
	option password '%s'
	option ip_source 'web'
	option ip_url 'http://ip.sb'
	option check_unit 'minutes'
	option check_interval '%d'
	option force_interval '%d'
	option interface 'wan'
	option bind_network 'wan'
`, shellSafe(p.ServiceID), shellSafe(p.Domain), shellSafe(p.ServiceName), shellSafe(p.Domain),
		shellSafe(p.Username), useIPv6, shellSafe(p.Password), p.CheckInterval, p.ForceInterval)

	f, err := os.OpenFile(opts.ConfigPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(block); err != nil {
		return err
	}
	_, _ = deps.RunCommand(opts.UpdaterPath, "-S", p.ServiceID, "start")
	return nil
}

func UpdateService(opts Options, deps Deps, serviceID string) error {
	opts = withDefaults(opts)
	deps = withDeps(deps)
	if strings.TrimSpace(serviceID) == "" {
		return fmt.Errorf("service id required")
	}
	_, err := deps.RunCommand(opts.UpdaterPath, "-S", serviceID, "start")
	return err
}

func ToggleService(opts Options, deps Deps, serviceID string) error {
	deps = withDeps(deps)
	enabled, _ := getUCI(deps, serviceID, "enabled")
	next := "1"
	if strings.TrimSpace(enabled) == "1" {
		next = "0"
	}
	if _, err := deps.RunCommand("uci", "set", fmt.Sprintf("ddns.%s.enabled=%s", serviceID, next)); err != nil {
		return err
	}
	_, err := deps.RunCommand("uci", "commit", "ddns")
	return err
}

func RemoveService(opts Options, deps Deps, serviceID string) error {
	deps = withDeps(deps)
	if _, err := deps.RunCommand("uci", "delete", "ddns."+serviceID); err != nil {
		return err
	}
	_, err := deps.RunCommand("uci", "commit", "ddns")
	return err
}

func PrintServiceLog(opts Options, serviceID string, out io.Writer) error {
	opts = withDefaults(opts)
	data, err := os.ReadFile(filepath.Join(opts.LogDir, serviceID+".log"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	_, err = out.Write(data)
	return err
}

func listServiceIDs(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`(?m)^\s*config\s+service\s+['"]?([^'"\s]+)['"]?\s*$`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out, nil
}

func listProviderNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			out = append(out, strings.Trim(parts[0], `"'`))
		}
	}
	return out, nil
}

func readLastRegisteredIP(logPath string) string {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`Registered IP[^']*'([^']+)'`)
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		m := re.FindStringSubmatch(lines[i])
		if len(m) >= 2 {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

func getUCI(deps Deps, serviceID string, key string) (string, error) {
	out, err := deps.RunCommand("uci", "get", fmt.Sprintf("ddns.%s.%s", serviceID, key))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func menuAdd(opts Options, deps Deps, reader *bufio.Reader) error {
	fmt.Fprintln(opts.Out, "1) IPV4")
	fmt.Fprintln(opts.Out, "2) IPV6")
	fmt.Fprint(opts.Out, "请选择网络模式> ")
	netChoice, err := readInt(reader)
	if err != nil {
		return err
	}
	useIPv6 := netChoice == 2

	servicesFile := opts.ServicesIPv4
	if useIPv6 {
		servicesFile = opts.ServicesIPv6
	}
	if _, err := os.Stat(servicesFile); err != nil {
		if _, err := os.Stat(opts.ServicesList); err == nil {
			servicesFile = opts.ServicesList
		} else {
			return fmt.Errorf("ddns service provider list missing")
		}
	}

	providers, err := listProviderNames(servicesFile)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return fmt.Errorf("empty ddns provider list")
	}
	fmt.Fprintln(opts.Out, "请选择服务提供商:")
	for i, p := range providers {
		fmt.Fprintf(opts.Out, "%d) %s\n", i+1, p)
	}
	fmt.Fprint(opts.Out, "请输入数字> ")
	pick, err := readInt(reader)
	if err != nil {
		return err
	}
	if pick < 1 || pick > len(providers) {
		return fmt.Errorf("invalid provider selection")
	}
	serviceName := providers[pick-1]
	serviceID := strings.ReplaceAll(serviceName, ".", "_")

	fmt.Fprint(opts.Out, "请输入你的域名> ")
	domain, err := readLine(reader)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Out, "请输入用户名或邮箱> ")
	username, err := readLine(reader)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Out, "请输入密码或令牌秘钥> ")
	password, err := readLine(reader)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Out, "请输入检测更新间隔(分钟, 默认10)> ")
	checkInterval := readIntOrDefault(reader, 10)
	fmt.Fprint(opts.Out, "请输入强制更新间隔(小时, 默认24)> ")
	forceInterval := readIntOrDefault(reader, 24)

	return AddService(opts, deps, AddParams{
		ServiceID:     serviceID,
		ServiceName:   serviceName,
		Domain:        domain,
		Username:      username,
		Password:      password,
		UseIPv6:       useIPv6,
		CheckInterval: checkInterval,
		ForceInterval: forceInterval,
	})
}

func menuServiceAction(opts Options, deps Deps, reader *bufio.Reader, serviceID string) error {
	for {
		fmt.Fprintln(opts.Out, "1) 立即更新")
		fmt.Fprintln(opts.Out, "2) 启用/停用")
		fmt.Fprintln(opts.Out, "3) 移除")
		fmt.Fprintln(opts.Out, "4) 查看日志")
		fmt.Fprintln(opts.Out, "0) 返回")
		fmt.Fprint(opts.Out, "请输入数字> ")
		n, err := readInt(reader)
		if err != nil {
			return err
		}
		switch n {
		case 0:
			return nil
		case 1:
			return UpdateService(opts, deps, serviceID)
		case 2:
			return ToggleService(opts, deps, serviceID)
		case 3:
			return RemoveService(opts, deps, serviceID)
		case 4:
			return PrintServiceLog(opts, serviceID, opts.Out)
		default:
			fmt.Fprintln(opts.Err, "输入错误")
		}
	}
}

func withDefaults(opts Options) Options {
	if opts.ConfigPath == "" {
		opts.ConfigPath = "/etc/config/ddns"
	}
	if opts.ServicesIPv4 == "" {
		opts.ServicesIPv4 = "/etc/ddns/services"
	}
	if opts.ServicesIPv6 == "" {
		opts.ServicesIPv6 = "/etc/ddns/services_ipv6"
	}
	if opts.ServicesList == "" {
		opts.ServicesList = "/usr/share/ddns/list"
	}
	if opts.LogDir == "" {
		opts.LogDir = "/var/log/ddns"
	}
	if opts.UpdaterPath == "" {
		opts.UpdaterPath = "/usr/lib/ddns/dynamic_dns_updater.sh"
	}
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}
	return opts
}

func withDeps(deps Deps) Deps {
	if deps.RunCommand == nil {
		deps.RunCommand = func(name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...)
			return cmd.CombinedOutput()
		}
	}
	return deps
}

func readLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func readInt(r *bufio.Reader) (int, error) {
	s, err := readLine(r)
	if err != nil {
		return 0, err
	}
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %q", s)
	}
	return n, nil
}

func readIntOrDefault(r *bufio.Reader, d int) int {
	s, _ := readLine(r)
	if s == "" {
		return d
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return n
}

func emptyAsDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func shellSafe(s string) string {
	return strings.ReplaceAll(s, "'", "")
}
