package toolsctl

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"shellcrash/internal/coreconfig"
	"shellcrash/internal/snapshotctl"
	"shellcrash/internal/startctl"
	"shellcrash/internal/taskctl"
)

type Options struct {
	CrashDir         string
	FirewallUserPath string
	DropbearKeyPath  string
	AuthKeysPath     string
	CrontabPath      string
	TmpDir           string
}

type sshState struct {
	Port    string
	Enabled bool
}

var toolsExecCommand = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

var toolsPortInUse = func(port int) bool {
	return isPortInUse(port)
}

var toolsSnapshotRun = func(opts snapshotctl.Options, deps snapshotctl.Deps) error {
	return snapshotctl.Run(opts, deps)
}

var toolsExecCmd = func(stdin io.Reader, out io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

var toolsCommandOutput = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

var toolsRunDebugMenu = func(opts Options, in io.Reader, out io.Writer) error {
	return RunDebugMenu(opts, in, out)
}

var toolsStartctlRun = func(crashDir string, args ...string) error {
	if strings.TrimSpace(crashDir) == "" {
		return fmt.Errorf("missing crashdir")
	}
	if len(args) == 0 {
		return fmt.Errorf("missing startctl action")
	}
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	action := strings.TrimSpace(args[0])
	extraArgs := append([]string(nil), args[1:]...)
	if action == "debug" {
		level := ""
		flash := false
		if len(extraArgs) > 0 {
			level = strings.TrimSpace(extraArgs[0])
		}
		if len(extraArgs) > 1 {
			flash = strings.TrimSpace(extraArgs[1]) == "flash"
		}
		return ctl.RunWithArgs(action, level, flash, nil)
	}
	return ctl.RunWithArgs(action, "", false, extraArgs)
}

var toolsRunTestCommandMenu = func(opts Options, in io.Reader, out io.Writer) error {
	return RunTestCommandMenu(opts, in, out)
}

var toolsRunLogPusherMenu = func(opts Options, in io.Reader, out io.Writer) error {
	return RunLogPusherMenu(opts, in, out)
}

var toolsRunSSHToolsMenu = func(opts Options, in io.Reader, out io.Writer) error {
	return RunSSHToolsMenu(opts, in, out)
}

var toolsRunUserguide = func(opts Options, in io.Reader, out io.Writer) error {
	return RunUserguide(opts, in, out)
}

var toolsTaskApplyRecommended = func(crashDir string) error {
	return taskctl.ApplyRecommendedTasks(crashDir)
}

var toolsCoreConfigRun = func(opts coreconfig.Options) (coreconfig.Result, error) {
	return coreconfig.Run(opts)
}

var toolsRunMiAutoSSHMenu = func(opts Options, in io.Reader, out io.Writer) error {
	return RunMiAutoSSH(opts, in, out)
}

var toolsToggleMiOta = func(opts Options) (string, error) {
	return ToggleMiOtaUpdate(opts)
}

var toolsToggleMiTun = func(opts Options) (string, error) {
	return ToggleMiTunfix(opts)
}

var toolsRunDDNSMenu = func(opts Options, in io.Reader, out io.Writer) error {
	candidates := []string{
		"shellcrash-ddnsctl",
		filepath.Join(opts.CrashDir, "bin", "shellcrash-ddnsctl"),
	}
	for _, bin := range candidates {
		if strings.Contains(bin, "/") {
			if !fileExists(bin) {
				continue
			}
		} else if !hasCommand(bin) {
			continue
		}
		cmd := exec.Command(bin, "--crashdir", opts.CrashDir, "menu")
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = out
		return cmd.Run()
	}
	if hasCommand("go") && fileExists(filepath.Join(opts.CrashDir, "go.mod")) {
		cmd := exec.Command("go", "run", filepath.Join(opts.CrashDir, "cmd", "ddnsctl"), "--crashdir", opts.CrashDir, "menu")
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = out
		return cmd.Run()
	}
	return fmt.Errorf("shellcrash-ddnsctl not found and go toolchain unavailable")
}

var toolsHasMiOtaBinary = func() bool {
	return fileExists("/usr/sbin/otapredownload")
}

func RunToolsMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		kv, err := loadCfgValues(opts)
		if err != nil {
			return err
		}
		systype := stripQuotes(kv["systype"])
		miAutoState := "未配置"
		if stripQuotes(kv["mi_mi_autoSSH"]) == "已配置" {
			miAutoState = "已配置"
		}
		miTunfix := "OFF"
		if fileExists(filepath.Join(opts.CrashDir, "tools", "tun.ko")) {
			miTunfix = "ON"
		}
		miUpdate := "启用"
		if hasActiveOtaCron(opts.CrontabPath) {
			miUpdate = "禁用"
		}

		fmt.Fprintln(out, "工具与优化")
		fmt.Fprintln(out, "1) ShellCrash测试菜单")
		fmt.Fprintln(out, "2) ShellCrash新手引导")
		fmt.Fprintln(out, "3) 日志及推送工具")
		if fileExists(opts.FirewallUserPath) {
			fmt.Fprintln(out, "4) 配置外网访问SSH")
		}
		if toolsHasMiOtaBinary() {
			fmt.Fprintf(out, "5) %s小米系统自动更新\n", miUpdate)
		}
		if systype == "mi_snapshot" {
			fmt.Fprintf(out, "6) 小米设备软固化SSH (%s)\n", miAutoState)
			fmt.Fprintf(out, "8) 小米设备Tun模块修复 (%s)\n", miTunfix)
		}
		fmt.Fprintln(out, "0) 返回上级菜单")
		fmt.Fprint(out, "请输入对应标号> ")

		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := toolsRunTestCommandMenu(opts, in, out); err != nil {
				return err
			}
		case "2":
			if err := toolsRunUserguide(opts, in, out); err != nil {
				return err
			}
			return nil
		case "3":
			if err := toolsRunLogPusherMenu(opts, in, out); err != nil {
				return err
			}
		case "4":
			if fileExists(opts.FirewallUserPath) {
				if err := toolsRunSSHToolsMenu(opts, in, out); err != nil {
					return err
				}
			}
		case "5":
			if toolsHasMiOtaBinary() {
				action, err := toolsToggleMiOta(opts)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "已%s小米路由器的自动更新，如未生效，请在官方APP中同步设置！\n", action)
			}
		case "6":
			if systype != "mi_snapshot" {
				fmt.Fprintln(out, "不支持的设备！")
				continue
			}
			if err := toolsRunMiAutoSSHMenu(opts, in, out); err != nil {
				return err
			}
		case "7":
			if err := toolsRunDDNSMenu(opts, in, out); err != nil {
				return err
			}
		case "8":
			if systype != "mi_snapshot" {
				fmt.Fprintln(out, "不支持的设备！")
				continue
			}
			action, err := toolsToggleMiTun(opts)
			if err != nil {
				return err
			}
			if action == "enabled" {
				fmt.Fprintln(out, "Tun模块补丁已启用，请重启服务。")
			} else {
				fmt.Fprintln(out, "Tun模块补丁已禁用，请重启设备。")
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunUserguide(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	restorePath := filepath.Join(opts.CrashDir, "configs.tar.gz")
	mode := ""
	for {
		fmt.Fprintln(out, "新手引导")
		fmt.Fprintln(out, "1) 我是旁路由/网关设备")
		fmt.Fprintln(out, "2) 我是本机代理设备")
		if fileExists(restorePath) {
			fmt.Fprintln(out, "3) 从备份恢复配置")
		}
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "1":
			if err := applyGatewayGuideProfile(opts); err != nil {
				return err
			}
			mode = "gateway"
			goto finalize
		case "2":
			if err := applyLocalProxyGuideProfile(opts); err != nil {
				return err
			}
			mode = "local"
			goto finalize
		case "3":
			if !fileExists(restorePath) {
				fmt.Fprintln(out, "未找到可恢复备份")
				continue
			}
			if err := restoreGuideBackup(opts.CrashDir, restorePath); err != nil {
				return err
			}
			fmt.Fprintln(out, "配置恢复成功")
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}

finalize:
	if mode == "gateway" && isIPForwardDisabled() {
		fmt.Fprintln(out, "检测到IP转发未开启，是否自动启用？")
		fmt.Fprint(out, "请输入(1=是/0=否)> ")
		res, err := readLine(reader)
		if err != nil {
			return err
		}
		if res == "1" {
			enableIPForward()
		}
	}

	if isLowDiskKB(opts.CrashDir, 10240) {
		fmt.Fprintln(out, "检测到可用空间较低，是否启用低内存运行模式？")
		fmt.Fprint(out, "请输入(1=是/0=否)> ")
		res, err := readLine(reader)
		if err != nil {
			return err
		}
		if res == "1" {
			if err := setCommandEnvValue(filepath.Join(opts.CrashDir, "configs", "command.env"), "BINDIR", "/tmp/ShellCrash"); err != nil {
				return err
			}
		}
	}

	if err := toolsTaskApplyRecommended(opts.CrashDir); err != nil {
		return err
	}

	if !fileExists(filepath.Join(opts.CrashDir, "yamls", "config.yaml")) && !fileExists(filepath.Join(opts.CrashDir, "jsons", "config.json")) {
		fmt.Fprintln(out, "当前未检测到配置文件")
		fmt.Fprintln(out, "1) 立即导入")
		fmt.Fprintln(out, "0) 暂不导入")
		fmt.Fprint(out, "请输入对应标号> ")
		res, err := readLine(reader)
		if err != nil {
			return err
		}
		if res == "1" {
			tmpDir := opts.TmpDir
			if cfg, cfgErr := startctl.LoadConfig(opts.CrashDir); cfgErr == nil && strings.TrimSpace(cfg.TmpDir) != "" {
				tmpDir = cfg.TmpDir
			}
			if _, err := toolsCoreConfigRun(coreconfig.Options{CrashDir: opts.CrashDir, TmpDir: tmpDir}); err != nil {
				return err
			}
		}
	}

	fmt.Fprintln(out, "引导配置完成，请返回主菜单启动服务。")
	return nil
}

func applyGatewayGuideProfile(opts Options) error {
	kv, err := loadCfgValues(opts)
	if err != nil {
		return err
	}

	redirMod := "Mix"
	cputype := stripQuotes(kv["cputype"])
	if strings.Contains(cputype, "linux") && strings.Contains(cputype, "mips") {
		if !hasTPROXYTarget() && toolsExecCommand("modprobe", "xt_TPROXY") != nil {
			redirMod = "Redir"
		} else {
			redirMod = "Tproxy"
		}
	}

	crashCore := stripQuotes(kv["crashcore"])
	if crashCore == "" {
		crashCore = "meta"
	}
	for key, value := range map[string]string{
		"crashcore":         crashCore,
		"redir_mod":         redirMod,
		"dns_mod":           "mix",
		"firewall_area":     "1",
		"cn_ip_route":       "ON",
		"autostart":         "enable",
		"mi_mi_autoSSH_pwd": stripQuotes(kv["mi_mi_autoSSH_pwd"]),
	} {
		if err := setCfgValue(opts, key, value); err != nil {
			return err
		}
	}
	if hasGlobalIPv6() {
		for _, key := range []string{"ipv6_redir", "ipv6_support", "ipv6_dns", "cn_ipv6_route"} {
			if err := setCfgValue(opts, key, "ON"); err != nil {
				return err
			}
		}
	}
	_ = os.Remove(filepath.Join(opts.CrashDir, ".dis_startup"))
	_ = setProcSysValue("/proc/sys/net/bridge/bridge-nf-call-iptables", "0")
	_ = setProcSysValue("/proc/sys/net/bridge/bridge-nf-call-ip6tables", "0")
	return nil
}

func applyLocalProxyGuideProfile(opts Options) error {
	kv, err := loadCfgValues(opts)
	if err != nil {
		return err
	}
	if err := setCfgValue(opts, "redir_mod", "Redir"); err != nil {
		return err
	}
	cputype := stripQuotes(kv["cputype"])
	if strings.Contains(cputype, "linux") && strings.Contains(cputype, "mips") {
		if err := setCfgValue(opts, "crashcore", "clash"); err != nil {
			return err
		}
	}
	if err := setCfgValue(opts, "common_ports", "OFF"); err != nil {
		return err
	}
	return setCfgValue(opts, "firewall_area", "2")
}

func hasTPROXYTarget() bool {
	b, err := os.ReadFile("/proc/net/ip_tables_targets")
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "TPROXY")
}

func hasGlobalIPv6() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.To4() != nil {
				continue
			}
			if ip.IsGlobalUnicast() && !ip.IsLinkLocalUnicast() {
				return true
			}
		}
	}
	return false
}

func isIPForwardDisabled() bool {
	b, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	return err == nil && strings.TrimSpace(string(b)) == "0"
}

func enableIPForward() {
	_ = appendLineIfMissing("/etc/sysctl.conf", "net.ipv4.ip_forward = 1")
	_ = setProcSysValue("/proc/sys/net/ipv4/ip_forward", "1")
}

func setProcSysValue(path, value string) error {
	return os.WriteFile(path, []byte(strings.TrimSpace(value)), 0o644)
}

func appendLineIfMissing(path, line string) error {
	current, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := strings.ReplaceAll(string(current), "\r\n", "\n")
	if strings.Contains(text, line) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(line + "\n")
	return err
}

func isLowDiskKB(path string, threshold int64) bool {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return false
	}
	avail := int64(st.Bavail) * int64(st.Bsize) / 1024
	return avail < threshold
}

func setCommandEnvValue(path, key, value string) error {
	lines, err := readLinesAllowMissing(path)
	if err != nil {
		return err
	}
	prefix := key + "="
	updated := make([]string, 0, len(lines)+1)
	found := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			updated = append(updated, prefix+value)
			found = true
			continue
		}
		updated = append(updated, line)
	}
	if !found {
		updated = append(updated, prefix+value)
	}
	return writeLines(path, updated)
}

func restoreGuideBackup(crashDir, tarGzPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	targetRoot := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return err
	}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := strings.TrimSpace(hdr.Name)
		if name == "" {
			continue
		}
		name = strings.TrimPrefix(name, "./")
		dst := filepath.Join(targetRoot, name)
		cleanRoot := filepath.Clean(targetRoot) + string(os.PathSeparator)
		cleanDst := filepath.Clean(dst)
		if !strings.HasPrefix(cleanDst+string(os.PathSeparator), cleanRoot) {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanDst, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(cleanDst), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(cleanDst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func hasActiveOtaCron(path string) bool {
	lines, err := readLinesAllowMissing(path)
	if err != nil {
		return false
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "otapredownload") && !strings.HasPrefix(trimmed, "#") {
			return true
		}
	}
	return false
}

func RunSSHToolsMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		st, err := loadSSHState(opts)
		if err != nil {
			return err
		}
		status := "开启"
		if st.Enabled {
			status = "禁止"
		}
		fmt.Fprintln(out, "外网访问SSH工具")
		fmt.Fprintf(out, "当前外网访问端口: %s\n", st.Port)
		fmt.Fprintf(out, "当前状态: %s\n", status)
		fmt.Fprintln(out, "1) 修改外网访问端口")
		fmt.Fprintln(out, "2) 修改SSH访问密码")
		fmt.Fprintf(out, "3) %s外网访问SSH\n", status)
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
			if err := handleSetPort(opts, st.Port, reader, out); err != nil {
				return err
			}
		case "2":
			if err := runPasswd(); err != nil {
				fmt.Fprintln(out, "执行passwd失败")
			}
		case "3":
			if st.Enabled {
				if err := disableSSHExternal(opts, st.Port); err != nil {
					return err
				}
				fmt.Fprintln(out, "已禁止外网访问SSH")
			} else {
				if err := enableSSHExternal(opts, st.Port); err != nil {
					return err
				}
				fmt.Fprintln(out, "已开启外网访问SSH功能")
			}
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunMiAutoSSH(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	fmt.Fprintln(out, "请输入需要还原的SSH密码（不影响当前密码）")
	fmt.Fprint(out, "请输入(回车可跳过)> ")
	pwd, err := readLine(reader)
	if err != nil {
		return err
	}
	if err := copyFileIfExists(opts.DropbearKeyPath, filepath.Join(opts.CrashDir, "configs", "dropbear_rsa_host_key")); err != nil {
		return err
	}
	if err := copyFileIfExists(opts.AuthKeysPath, filepath.Join(opts.CrashDir, "configs", "authorized_keys")); err != nil {
		return err
	}
	if hasCommand("nvram") {
		_ = toolsExecCommand("nvram", "set", "ssh_en=1")
		_ = toolsExecCommand("nvram", "set", "telnet_en=1")
		_ = toolsExecCommand("nvram", "set", "uart_en=1")
		_ = toolsExecCommand("nvram", "set", "boot_wait=on")
		_ = toolsExecCommand("nvram", "commit")
	}
	if err := setCfgValue(opts, "mi_mi_autoSSH", "已配置"); err != nil {
		return err
	}
	if err := setCfgValue(opts, "mi_mi_autoSSH_pwd", pwd); err != nil {
		return err
	}
	fmt.Fprintln(out, "设置成功！")
	return nil
}

func RunTestCommandMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintln(out, "这里是测试命令菜单")
		fmt.Fprintln(out, "1) Debug模式运行内核")
		fmt.Fprintln(out, "2) 查看系统DNS端口(:53)占用")
		fmt.Fprintln(out, "3) 测试ssl加密(aes-128-gcm)跑分")
		fmt.Fprintln(out, "4) 查看ShellCrash相关路由规则")
		fmt.Fprintln(out, "5) 查看内核配置文件前40行")
		fmt.Fprintln(out, "6) 测试代理服务器连通性(google.tw)")
		fmt.Fprintln(out, "0) 返回上级目录")
		fmt.Fprint(out, "请输入对应数字> ")

		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := toolsRunDebugMenu(opts, in, out); err != nil {
				fmt.Fprintf(out, "debug菜单执行失败: %v\n", err)
			}
		case "2":
			fmt.Fprintln(out, "===========================================================")
			if err := runShowDNSPortUsage(out); err != nil {
				fmt.Fprintf(out, "查询失败: %v\n", err)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "可以使用 netstat -ntulp |grep xxx 来查询任意(xxx)端口")
			fmt.Fprintln(out, "===========================================================")
		case "3":
			_ = toolsExecCmd(nil, out, "openssl", "speed", "-multi", "4", "-evp", "aes-128-gcm")
		case "4":
			runShowRouteRules(opts, out)
		case "5":
			if err := runShowCoreConfig(opts, out); err != nil {
				fmt.Fprintf(out, "读取配置失败: %v\n", err)
			}
		case "6":
			runProxyConnectivityTest(opts, out)
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunDebugMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	cfg, err := startctl.LoadConfig(opts.CrashDir)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(in)
	for {
		configTmp := filepath.Join(cfg.TmpDir, "config.yaml")
		if strings.Contains(cfg.CrashCore, "singbox") {
			configTmp = filepath.Join(cfg.TmpDir, "jsons")
		}
		fmt.Fprintln(out, "注意：Debug运行均会停止原本的内核服务")
		fmt.Fprintf(out, "后台运行日志地址：%s\n", filepath.Join(cfg.TmpDir, "debug.log"))
		fmt.Fprintln(out, "如长时间运行后台监测，日志等级推荐error！防止文件过大！")
		fmt.Fprintln(out, "你亦可通过：crash -s debug warning 命令使用其他日志等级")
		fmt.Fprintf(out, "1) 仅测试%s配置文件可用性\n", configTmp)
		fmt.Fprintf(out, "2) 前台运行%s配置文件，不配置防火墙劫持(Ctrl+C手动停止)\n", configTmp)
		fmt.Fprintln(out, "3) 后台运行完整启动流程并配置防火墙劫持，日志等级：error")
		fmt.Fprintln(out, "4) 后台运行完整启动流程并配置防火墙劫持，日志等级：info")
		fmt.Fprintln(out, "5) 后台运行完整启动流程并配置防火墙劫持，日志等级：debug")
		fmt.Fprintf(out, "6) 后台运行完整启动流程并配置防火墙劫持，且将错误日志打印到闪存：%s\n", filepath.Join(opts.CrashDir, "debug.log"))
		fmt.Fprintln(out, "8) 后台运行完整启动流程,输出执行错误并查找上下文，之后关闭进程")
		if fileExists(filepath.Join(cfg.TmpDir, "jsons", "inbounds.json")) {
			fmt.Fprintf(out, "9) 将%s下json文件合并为%s\n", configTmp, filepath.Join(cfg.TmpDir, "debug.json"))
		}
		fmt.Fprintln(out, "0) 返回上级目录")
		fmt.Fprint(out, "请输入对应标号> ")

		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := debugConfigCheck(cfg, out); err != nil {
				fmt.Fprintf(out, "测试失败: %v\n", err)
			}
			return nil
		case "2":
			if err := debugForegroundRun(cfg, out); err != nil {
				fmt.Fprintf(out, "前台运行失败: %v\n", err)
			}
			return nil
		case "3":
			return toolsStartctlRun(opts.CrashDir, "debug", "error")
		case "4":
			return toolsStartctlRun(opts.CrashDir, "debug", "info")
		case "5":
			return toolsStartctlRun(opts.CrashDir, "debug", "debug")
		case "6":
			fmt.Fprintln(out, "频繁写入闪存会导致闪存寿命降低，如无必要请勿使用。")
			fmt.Fprint(out, "是否确认启用此功能？(1=是/0=否)> ")
			res, err := readLine(reader)
			if err != nil {
				return err
			}
			if res == "1" {
				return toolsStartctlRun(opts.CrashDir, "debug", "debug", "flash")
			}
		case "8":
			return toolsStartctlRun(opts.CrashDir, "debug")
		case "9":
			if !fileExists(filepath.Join(cfg.TmpDir, "jsons", "inbounds.json")) {
				fmt.Fprintln(out, "未检测到可合并的json分片")
				continue
			}
			if err := debugMergeJSON(cfg, out); err != nil {
				fmt.Fprintf(out, "合并失败: %v\n", err)
				continue
			}
			fmt.Fprintln(out, "合并成功！")
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func debugConfigCheck(cfg startctl.Config, out io.Writer) error {
	_ = toolsStartctlRun(cfg.CrashDir, "stop")
	if err := toolsStartctlRun(cfg.CrashDir, "bfstart"); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(cfg.TmpDir, "CrashCore"))

	coreBin := filepath.Join(cfg.TmpDir, "CrashCore")
	if strings.Contains(cfg.CrashCore, "singbox") {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, coreBin, "run", "-D", cfg.BinDir, "-C", filepath.Join(cfg.TmpDir, "jsons"))
		cmd.Stdout = out
		cmd.Stderr = out
		err := cmd.Run()
		if err != nil && ctx.Err() == context.DeadlineExceeded {
			return nil
		}
		return err
	}
	return toolsExecCmd(nil, out, coreBin, "-t", "-d", cfg.BinDir, "-f", filepath.Join(cfg.TmpDir, "config.yaml"))
}

func debugForegroundRun(cfg startctl.Config, out io.Writer) error {
	_ = toolsStartctlRun(cfg.CrashDir, "stop")
	if err := toolsStartctlRun(cfg.CrashDir, "bfstart"); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(cfg.TmpDir, "CrashCore"))

	cmdText := strings.TrimSpace(cfg.Command)
	if cmdText == "" {
		return fmt.Errorf("COMMAND is empty")
	}
	command, args, err := splitCommandLine(cmdText)
	if err != nil {
		return err
	}
	cmd := exec.Command(command, args...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+cfg.CrashDir,
		"BINDIR="+cfg.BinDir,
		"TMPDIR="+cfg.TmpDir,
	)
	return cmd.Run()
}

func debugMergeJSON(cfg startctl.Config, out io.Writer) error {
	coreBin := filepath.Join(cfg.TmpDir, "CrashCore")
	if !fileExists(coreBin) {
		if err := toolsStartctlRun(cfg.CrashDir, "bfstart"); err != nil {
			return err
		}
	}
	return toolsExecCmd(nil, out, coreBin, "merge", filepath.Join(cfg.TmpDir, "debug.json"), "-C", filepath.Join(cfg.TmpDir, "jsons"))
}

func runShowRouteRules(opts Options, out io.Writer) {
	cfg, err := startctl.LoadConfig(opts.CrashDir)
	if err != nil {
		fmt.Fprintf(out, "读取配置失败: %v\n", err)
		return
	}
	if cfg.FirewallMod == "nftables" {
		data, outputErr := toolsCommandOutput("nft", "list", "table", "inet", "shellcrash")
		if outputErr != nil {
			_ = toolsExecCmd(nil, out, "nft", "list", "table", "inet", "shellcrash")
			return
		}
		filtered := filterNftRouteOutput(string(data))
		if strings.TrimSpace(filtered) == "" {
			return
		}
		if !strings.HasSuffix(filtered, "\n") {
			filtered += "\n"
		}
		_, _ = io.WriteString(out, filtered)
		return
	}
	if cfg.FirewallArea == "1" || cfg.FirewallArea == "3" || cfg.FirewallArea == "5" {
		fmt.Fprintln(out, "----------------Redir+DNS---------------------")
		_ = toolsExecCmd(nil, out, "iptables", "-t", "nat", "-L", "PREROUTING", "--line-numbers")
		_ = toolsExecCmd(nil, out, "iptables", "-t", "nat", "-L", "shellcrash_dns", "--line-numbers")
		if strings.Contains(cfg.RedirMod, "Redir") || strings.Contains(cfg.RedirMod, "Mix") {
			_ = toolsExecCmd(nil, out, "iptables", "-t", "nat", "-L", "shellcrash", "--line-numbers")
		}
		if strings.Contains(cfg.RedirMod, "Tproxy") || strings.Contains(cfg.RedirMod, "Mix") || strings.Contains(cfg.RedirMod, "Tun") {
			fmt.Fprintln(out, "----------------Tun/Tproxy-------------------")
			_ = toolsExecCmd(nil, out, "iptables", "-t", "mangle", "-L", "PREROUTING", "--line-numbers")
			_ = toolsExecCmd(nil, out, "iptables", "-t", "mangle", "-L", "shellcrash_mark", "--line-numbers")
		}
	}
	if cfg.FirewallArea == "2" || cfg.FirewallArea == "3" {
		fmt.Fprintln(out, "-------------OUTPUT-Redir+DNS----------------")
		_ = toolsExecCmd(nil, out, "iptables", "-t", "nat", "-L", "OUTPUT", "--line-numbers")
		_ = toolsExecCmd(nil, out, "iptables", "-t", "nat", "-L", "shellcrash_dns_out", "--line-numbers")
		if strings.Contains(cfg.RedirMod, "Redir") || strings.Contains(cfg.RedirMod, "Mix") {
			_ = toolsExecCmd(nil, out, "iptables", "-t", "nat", "-L", "shellcrash_out", "--line-numbers")
		}
		if strings.Contains(cfg.RedirMod, "Tproxy") || strings.Contains(cfg.RedirMod, "Mix") || strings.Contains(cfg.RedirMod, "Tun") {
			fmt.Fprintln(out, "------------OUTPUT-Tun/Tproxy---------------")
			_ = toolsExecCmd(nil, out, "iptables", "-t", "mangle", "-L", "OUTPUT", "--line-numbers")
			_ = toolsExecCmd(nil, out, "iptables", "-t", "mangle", "-L", "shellcrash_mark_out", "--line-numbers")
		}
	}
	fmt.Fprintln(out, "----------------本机防火墙---------------------")
	_ = toolsExecCmd(nil, out, "iptables", "-L", "INPUT", "--line-numbers")
}

func runShowDNSPortUsage(out io.Writer) error {
	data, err := toolsCommandOutput("netstat", "-ntulp")
	if err != nil {
		return err
	}
	for _, raw := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, " \t")
		if strings.Contains(line, "53") {
			fmt.Fprintln(out, line)
		}
	}
	return nil
}

func filterNftRouteOutput(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	skipSet := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if skipSet {
			if strings.Contains(trimmed, "}") {
				skipSet = false
			}
			continue
		}
		if strings.Contains(line, "set cn_ip {") || strings.Contains(line, "set cn_ip6 {") {
			if !strings.Contains(trimmed, "}") {
				skipSet = true
			}
			continue
		}
		if trimmed == "}" {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func splitCommandLine(s string) (string, []string, error) {
	text := strings.TrimSpace(s)
	if text == "" {
		return "", nil, fmt.Errorf("COMMAND is empty")
	}
	args := make([]string, 0, 8)
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		args = append(args, cur.String())
		cur.Reset()
	}
	for _, r := range text {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case unicode.IsSpace(r) && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if escaped || inSingle || inDouble {
		return "", nil, fmt.Errorf("invalid COMMAND quoting")
	}
	flush()
	if len(args) == 0 {
		return "", nil, fmt.Errorf("COMMAND is empty")
	}
	return args[0], args[1:], nil
}

func runShowCoreConfig(opts Options, out io.Writer) error {
	cfg, err := startctl.LoadConfig(opts.CrashDir)
	if err != nil {
		return err
	}
	path := filepath.Join(opts.CrashDir, "yamls", "config.yaml")
	if strings.Contains(cfg.CrashCore, "singbox") {
		path = filepath.Join(opts.CrashDir, "jsons", "config.json")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	max := 40
	if len(lines) < max {
		max = len(lines)
	}
	for i := 0; i < max; i++ {
		fmt.Fprintln(out, lines[i])
	}
	return nil
}

func runProxyConnectivityTest(opts Options, out io.Writer) {
	cfg, err := startctl.LoadConfig(opts.CrashDir)
	if err != nil {
		fmt.Fprintf(out, "读取配置失败: %v\n", err)
		return
	}
	port := cfg.MixPort
	if strings.TrimSpace(port) == "" {
		port = "7890"
	}
	target := "127.0.0.1:" + strings.TrimSpace(port)
	client := &net.Dialer{Timeout: 3 * time.Second}
	start := time.Now()
	conn, err := client.Dial("tcp", target)
	if err != nil {
		fmt.Fprintln(out, "连接超时！请重试或检查节点配置！")
		return
	}
	_ = conn.Close()
	delay := time.Since(start).Milliseconds()
	fmt.Fprintf(out, "连接成功！响应时间约为：%d ms\n", delay)
}

func withDefaults(opts Options) Options {
	if strings.TrimSpace(opts.CrashDir) == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	if strings.TrimSpace(opts.FirewallUserPath) == "" {
		opts.FirewallUserPath = "/etc/firewall.user"
	}
	if strings.TrimSpace(opts.DropbearKeyPath) == "" {
		opts.DropbearKeyPath = "/etc/dropbear/dropbear_rsa_host_key"
	}
	if strings.TrimSpace(opts.AuthKeysPath) == "" {
		opts.AuthKeysPath = "/etc/dropbear/authorized_keys"
	}
	if strings.TrimSpace(opts.CrontabPath) == "" {
		opts.CrontabPath = "/etc/crontabs/root"
	}
	if strings.TrimSpace(opts.TmpDir) == "" {
		opts.TmpDir = os.TempDir()
	}
	return opts
}

func RunLogPusherMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		kv, err := loadCfgValues(opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "日志及推送工具")
		fmt.Fprintf(out, "1) Telegram推送   —— %s\n", onOff(kv["push_TG"]))
		fmt.Fprintf(out, "2) PushDeer推送   —— %s\n", onOff(kv["push_Deer"]))
		fmt.Fprintf(out, "3) Bark推送-IOS   —— %s\n", onOff(kv["push_bark"]))
		fmt.Fprintf(out, "4) Passover推送   —— %s\n", onOff(kv["push_Po"]))
		fmt.Fprintf(out, "5) PushPlus推送   —— %s\n", onOff(kv["push_PP"]))
		fmt.Fprintf(out, "6) SynoChat推送   —— %s\n", onOff(kv["push_SynoChat"]))
		fmt.Fprintf(out, "7) Gotify推送     —— %s\n", onOff(kv["push_Gotify"]))
		fmt.Fprintln(out, "a) 查看运行日志")
		fmt.Fprintf(out, "b) 推送任务日志   —— %s\n", onOffBool(kv["task_push"] == "1"))
		device := stripQuotes(kv["device_name"])
		if device == "" {
			device = "未设置"
		}
		fmt.Fprintf(out, "c) 设置设备名称   —— %s\n", device)
		fmt.Fprintln(out, "d) 清空日志文件")
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
			if err := handleProviderToggle(opts, kv, reader, out, "push_TG", "Telegram推送", true, nil); err != nil {
				return err
			}
		case "2":
			if err := handleProviderToggle(opts, kv, reader, out, "push_Deer", "PushDeer推送", false, nil); err != nil {
				return err
			}
		case "3":
			if err := handleProviderToggle(opts, kv, reader, out, "push_bark", "Bark推送", false, []string{"bark_param"}); err != nil {
				return err
			}
		case "4":
			if err := handlePushover(opts, kv, reader, out); err != nil {
				return err
			}
		case "5":
			if err := handleProviderToggle(opts, kv, reader, out, "push_PP", "PushPlus推送", false, nil); err != nil {
				return err
			}
		case "6":
			if err := handleSynoChat(opts, kv, reader, out); err != nil {
				return err
			}
		case "7":
			if err := handleProviderToggle(opts, kv, reader, out, "push_Gotify", "Gotify推送", false, nil); err != nil {
				return err
			}
		case "a":
			if err := showLogFile(filepath.Join(opts.TmpDir, "ShellCrash.log"), out); err != nil {
				return err
			}
		case "b":
			next := "1"
			if stripQuotes(kv["task_push"]) == "1" {
				next = ""
			}
			if err := setCfgValue(opts, "task_push", next); err != nil {
				return err
			}
		case "c":
			fmt.Fprint(out, "请输入设备名称(回车跳过)> ")
			name, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) != "" {
				if err := setCfgValue(opts, "device_name", strings.TrimSpace(name)); err != nil {
					return err
				}
			}
		case "d":
			if err := os.Remove(filepath.Join(opts.TmpDir, "ShellCrash.log")); err != nil && !os.IsNotExist(err) {
				return err
			}
			fmt.Fprintln(out, "运行日志及任务日志均已清空")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func showLogFile(path string, out io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "未找到相关日志")
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		fmt.Fprintln(out, "未找到相关日志")
		return nil
	}
	fmt.Fprintln(out, string(b))
	return nil
}

func handlePushover(opts Options, kv map[string]string, reader *bufio.Reader, out io.Writer) error {
	if stripQuotes(kv["push_Po"]) != "" {
		if !confirmDisable(reader, out, "Pushover推送") {
			return nil
		}
		if err := setCfgValue(opts, "push_Po", ""); err != nil {
			return err
		}
		return setCfgValue(opts, "push_Po_key", "")
	}
	fmt.Fprint(out, "请输入Pushover User Key(0返回)> ")
	key, err := readLine(reader)
	if err != nil {
		return err
	}
	if key == "0" || strings.TrimSpace(key) == "" {
		return nil
	}
	fmt.Fprint(out, "请输入Pushover API Token> ")
	token, err := readLine(reader)
	if err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	if err := setCfgValue(opts, "push_Po", token); err != nil {
		return err
	}
	if err := setCfgValue(opts, "push_Po_key", strings.TrimSpace(key)); err != nil {
		return err
	}
	fmt.Fprintln(out, "已完成Pushover日志推送设置")
	return nil
}

func handleSynoChat(opts Options, kv map[string]string, reader *bufio.Reader, out io.Writer) error {
	if stripQuotes(kv["push_SynoChat"]) != "" {
		if !confirmDisable(reader, out, "SynoChat推送") {
			return nil
		}
		for _, k := range []string{"push_SynoChat", "push_ChatURL", "push_ChatTOKEN", "push_ChatUSERID"} {
			if err := setCfgValue(opts, k, ""); err != nil {
				return err
			}
		}
		return nil
	}
	fmt.Fprint(out, "请输入Synology DSM主页地址(0返回)> ")
	url, err := readLine(reader)
	if err != nil {
		return err
	}
	if url == "0" || strings.TrimSpace(url) == "" {
		return nil
	}
	fmt.Fprint(out, "请输入Synology Chat Token> ")
	token, err := readLine(reader)
	if err != nil {
		return err
	}
	fmt.Fprint(out, "请输入user_id> ")
	userID, err := readLine(reader)
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"push_SynoChat":   strings.TrimSpace(userID),
		"push_ChatURL":    strings.TrimSpace(url),
		"push_ChatTOKEN":  strings.TrimSpace(token),
		"push_ChatUSERID": strings.TrimSpace(userID),
	} {
		if err := setCfgValue(opts, k, v); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "已完成SynoChat日志推送设置")
	return nil
}

func handleProviderToggle(opts Options, kv map[string]string, reader *bufio.Reader, out io.Writer, key, title string, needsChatID bool, extraClear []string) error {
	if stripQuotes(kv[key]) != "" {
		if !confirmDisable(reader, out, title) {
			return nil
		}
		if err := setCfgValue(opts, key, ""); err != nil {
			return err
		}
		if needsChatID {
			if err := setCfgValue(opts, "chat_ID", ""); err != nil {
				return err
			}
		}
		for _, clearKey := range extraClear {
			if err := setCfgValue(opts, clearKey, ""); err != nil {
				return err
			}
		}
		return nil
	}
	if needsChatID {
		fmt.Fprint(out, "请输入Telegram Bot Token(0返回)> ")
		token, err := readLine(reader)
		if err != nil {
			return err
		}
		token = strings.TrimSpace(token)
		if token == "0" || token == "" {
			return nil
		}
		fmt.Fprint(out, "请输入chat_id> ")
		chatID, err := readLine(reader)
		if err != nil {
			return err
		}
		chatID = strings.TrimSpace(chatID)
		if chatID == "" {
			fmt.Fprintln(out, "输入错误")
			return nil
		}
		if err := setCfgValue(opts, key, token); err != nil {
			return err
		}
		return setCfgValue(opts, "chat_ID", chatID)
	}
	fmt.Fprintf(out, "请输入%s参数(0返回)> ", title)
	value, err := readLine(reader)
	if err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	if value == "0" || value == "" {
		return nil
	}
	return setCfgValue(opts, key, value)
}

func confirmDisable(reader *bufio.Reader, out io.Writer, title string) bool {
	fmt.Fprintf(out, "是否确认关闭%s？(1=是/0=否)> ", title)
	res, err := readLine(reader)
	if err != nil {
		return false
	}
	return res == "1"
}

func onOff(raw string) string {
	if stripQuotes(raw) == "" {
		return "OFF"
	}
	return "ON"
}

func onOffBool(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func loadCfgValues(opts Options) (map[string]string, error) {
	path := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	kv, err := parseKVFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	return kv, nil
}

func ToggleMiOtaUpdate(opts Options) (string, error) {
	opts = withDefaults(opts)
	const enabledLine = "15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1"
	const disabledLine = "#15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1"

	lines, err := readLinesAllowMissing(opts.CrontabPath)
	if err != nil {
		return "", err
	}

	hasActive := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "otapredownload") && !strings.HasPrefix(trimmed, "#") {
			hasActive = true
			break
		}
	}

	action := "启用"
	if hasActive {
		action = "禁用"
		updated, changed := toggleOtaLines(lines, false)
		if !changed {
			updated = append(updated, disabledLine)
		}
		if err := writeLines(opts.CrontabPath, updated); err != nil {
			return "", err
		}
		return action, nil
	}

	updated, changed := toggleOtaLines(lines, true)
	if !changed {
		updated = append(updated, enabledLine)
	}
	if err := writeLines(opts.CrontabPath, updated); err != nil {
		return "", err
	}
	return action, nil
}

func ToggleMiTunfix(opts Options) (string, error) {
	opts = withDefaults(opts)
	target := filepath.Join(opts.CrashDir, "tools", "tun.ko")
	if fileExists(target) {
		if err := os.Remove(target); err != nil {
			return "", err
		}
		return "disabled", nil
	}

	if !hasCommand("modinfo") {
		return "", fmt.Errorf("modinfo command is unavailable")
	}
	out, err := exec.Command("modinfo", "tun").CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return "", fmt.Errorf("tun module already exists, skip patch")
	}

	src := filepath.Join(opts.CrashDir, "bin", "fix", "tun.ko")
	if !fileExists(src) {
		return "", fmt.Errorf("tun patch file not found: %s", src)
	}
	if err := copyFileIfExists(src, target); err != nil {
		return "", err
	}
	if err := toolsSnapshotRun(snapshotctl.Options{
		CrashDir: opts.CrashDir,
		Action:   "tunfix",
	}, snapshotctl.Deps{}); err != nil {
		return "", err
	}
	return "enabled", nil
}

func toggleOtaLines(lines []string, enable bool) ([]string, bool) {
	updated := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if !strings.Contains(line, "otapredownload") {
			updated = append(updated, line)
			continue
		}
		trimmed := strings.TrimSpace(line)
		if enable {
			if strings.HasPrefix(trimmed, "#") {
				uncommented := strings.TrimLeft(line, " \t")
				if strings.HasPrefix(uncommented, "#") {
					uncommented = strings.TrimPrefix(uncommented, "#")
				}
				updated = append(updated, uncommented)
				changed = true
				continue
			}
		} else {
			if !strings.HasPrefix(trimmed, "#") {
				updated = append(updated, "#"+line)
				changed = true
				continue
			}
		}
		updated = append(updated, line)
	}
	return updated, changed
}

func readLinesAllowMissing(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	content := strings.ReplaceAll(string(b), "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil, nil
	}
	return strings.Split(content, "\n"), nil
}

func writeLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func handleSetPort(opts Options, oldPort string, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprint(out, "请输入端口号(1000-65535)> ")
	raw, err := readLine(reader)
	if err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1000 || n > 65535 {
		fmt.Fprintln(out, "输入错误！请输入正确的数值(1000-65535)")
		return nil
	}
	if toolsPortInUse(n) {
		fmt.Fprintln(out, "当前端口已被其他进程占用，请重新输入")
		return nil
	}
	if err := setCfgValue(opts, "ssh_port", strconv.Itoa(n)); err != nil {
		return err
	}
	if err := disableSSHExternal(opts, oldPort); err != nil {
		return err
	}
	fmt.Fprintln(out, "设置成功，请重新开启外网访问SSH功能")
	return nil
}

func runPasswd() error {
	cmd := exec.Command("passwd")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func enableSSHExternal(opts Options, port string) error {
	_ = toolsExecCommand("iptables", "-w", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "-m", "multiport", "--dports", port, "-j", "REDIRECT", "--to-ports", "22")
	if hasCommand("ip6tables") {
		_ = toolsExecCommand("ip6tables", "-w", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "-m", "multiport", "--dports", port, "-j", "REDIRECT", "--to-ports", "22")
	}
	lines := []string{
		fmt.Sprintf("iptables -w -t nat -A PREROUTING -p tcp -m multiport --dports %s -j REDIRECT --to-ports 22 #启用外网访问SSH服务", port),
	}
	if hasCommand("ip6tables") {
		lines = append(lines, fmt.Sprintf("ip6tables -w -t nat -A PREROUTING -p tcp -m multiport --dports %s -j REDIRECT --to-ports 22 #启用外网访问SSH服务", port))
	}
	return appendLines(opts.FirewallUserPath, lines)
}

func disableSSHExternal(opts Options, port string) error {
	_ = toolsExecCommand("iptables", "-w", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-m", "multiport", "--dports", port, "-j", "REDIRECT", "--to-ports", "22")
	if hasCommand("ip6tables") {
		_ = toolsExecCommand("ip6tables", "-w", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-m", "multiport", "--dports", port, "-j", "REDIRECT", "--to-ports", "22")
	}
	return removeMarkedFirewallLines(opts.FirewallUserPath)
}

func loadSSHState(opts Options) (sshState, error) {
	state := sshState{Port: "10022"}
	kv, err := parseKVFile(filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg"))
	if err != nil && !os.IsNotExist(err) {
		return state, err
	}
	if v := stripQuotes(kv["ssh_port"]); v != "" {
		state.Port = v
	}
	b, err := os.ReadFile(opts.FirewallUserPath)
	if err == nil && strings.Contains(string(b), "启用外网访问SSH服务") {
		state.Enabled = true
	}
	return state, nil
}

func removeMarkedFirewallLines(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "启用外网访问SSH服务") {
			continue
		}
		out = append(out, line)
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

func appendLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func copyFileIfExists(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isPortInUse(port int) bool {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
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
	kv[key] = value
	return writeKVFile(path, kv)
}

func readLine(reader *bufio.Reader) (string, error) {
	raw, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(raw), nil
}

func stripQuotes(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func parseKVFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	lines := strings.Split(string(b), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		out[k] = v
	}
	return out, nil
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
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(values[k])
		sb.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}
