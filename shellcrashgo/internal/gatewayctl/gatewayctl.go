package gatewayctl

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"shellcrash/internal/ddnsctl"
	"shellcrash/internal/firewall"
	"shellcrash/internal/startctl"
)

type Options struct {
	CrashDir string
}

type State struct {
	CrashCore      string
	FirewallMod    string
	FWWan          string
	FWWanPorts     string
	CommonPorts    string
	QuicRJ         string
	CNIPRoute      string
	MacFilterType  string
	CustHostIPv4   string
	ReplaceHostV4  string
	ReserveIPv4    string
	Multiport      string
	MixPort        int
	DBPort         int
	TSService      string
	TSAuthKey      string
	TSSubnet       bool
	TSExitNode     bool
	TSHostname     string
	WGService      string
	WGServer       string
	WGPort         string
	WGPublicKey    string
	WGPreSharedKey string
	WGPrivateKey   string
	WGIPv4         string
	WGIPv6         string
	VMSPort        string
	SSSPort        string
	VMSService     string
	VMSWSPath      string
	VMSUUID        string
	VMSHost        string
	SSSService     string
	SSSCipher      string
	SSSPwd         string
	BotTGService   string
	TGMenuPush     string
	TGChatID       string
}

var gatewayHTTPDo = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	return client.Do(req)
}

var gatewayRunControllerAction = func(crashDir string, action string) error {
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	return ctl.RunWithArgs(action, "", false, nil)
}

var gatewayIsProcessRunning = func(name string) bool {
	return exec.Command("pidof", name).Run() == nil
}
var gatewayHasIPSet = func() bool {
	return exec.Command("ipset", "-v").Run() == nil
}

var gatewayRunDDNSMenu = func(in io.Reader, out io.Writer) error {
	return ddnsctl.RunMenu(ddnsctl.Options{In: in, Out: out, Err: out}, ddnsctl.Deps{})
}

var vmessUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var macPattern = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

func RunGatewayMenu(opts Options, in io.Reader, out io.Writer) error {
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
		fmt.Fprintln(out, "访问与控制菜单")
		fmt.Fprintf(out, "1) 配置公网访问防火墙 [%s]\n", st.FWWan)
		fmt.Fprintf(out, "2) 配置Telegram专属控制机器人 [%s]\n", st.BotTGService)
		fmt.Fprintln(out, "3) 配置DDNS自动域名")
		fmt.Fprintf(out, "4) 自定义公网Vmess入站节点 [%s]\n", st.VMSService)
		fmt.Fprintf(out, "5) 自定义公网ShadowSocks入站节点 [%s]\n", st.SSSService)
		if strings.Contains(st.CrashCore, "sing") {
			fmt.Fprintf(out, "6) 配置Tailscale内网穿透 [%s]\n", st.TSService)
			fmt.Fprintf(out, "7) 配置Wireguard客户端 [%s]\n", st.WGService)
		} else {
			fmt.Fprintln(out, "6) 配置Tailscale内网穿透 [仅Singbox核心可用]")
			fmt.Fprintln(out, "7) 配置Wireguard客户端 [仅Singbox核心可用]")
		}
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
			if gatewayIsProcessRunning("CrashCore") && st.FirewallMod == "iptables" {
				fmt.Fprintln(out, "公网访问防火墙需要先停止服务，是否确认继续？")
				fmt.Fprintln(out, "1) 是")
				fmt.Fprintln(out, "0) 否")
				fmt.Fprint(out, "请输入对应标号> ")
				confirm, err := readLine(reader)
				if err != nil {
					return err
				}
				if confirm != "1" {
					continue
				}
				if err := gatewayRunControllerAction(opts.CrashDir, "stop"); err != nil {
					return err
				}
			}
			if err := RunFirewallMenu(opts, reader, out); err != nil {
				return err
			}
		case "2":
			if err := RunTGMenu(opts, reader, out); err != nil {
				return err
			}
		case "3":
			if err := gatewayRunDDNSMenu(reader, out); err != nil {
				return err
			}
		case "4":
			if err := RunVmessMenu(opts, reader, out); err != nil {
				return err
			}
		case "5":
			if err := RunShadowsocksMenu(opts, reader, out); err != nil {
				return err
			}
		case "6":
			if !strings.Contains(st.CrashCore, "sing") {
				fmt.Fprintf(out, "%s内核暂不支持此功能，请先更换内核\n", st.CrashCore)
				continue
			}
			if err := RunTailscaleMenu(opts, reader, out); err != nil {
				return err
			}
		case "7":
			if !strings.Contains(st.CrashCore, "sing") {
				fmt.Fprintf(out, "%s内核暂不支持此功能，请先更换内核\n", st.CrashCore)
				continue
			}
			if err := RunWireGuardMenu(opts, reader, out); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunTGMenu(opts Options, in io.Reader, out io.Writer) error {
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
		bindState := "未绑定"
		if strings.TrimSpace(st.TGChatID) != "" {
			bindState = "已绑定"
		}
		fmt.Fprintln(out, "Telegram专属控制机器人")
		fmt.Fprintf(out, "1) 启用/关闭TG-BOT服务 [%s]\n", st.BotTGService)
		fmt.Fprintf(out, "2) TG-BOT绑定设置 [%s]\n", bindState)
		fmt.Fprintf(out, "3) 启动时推送菜单 [%s]\n", st.TGMenuPush)
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
			if strings.TrimSpace(st.TGChatID) == "" {
				fmt.Fprintln(out, "请先绑定TG-BOT")
				continue
			}
			next, err := SetTGService(opts, "toggle")
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "TG-BOT服务状态: %s\n", next)
		case "2":
			fmt.Fprint(out, "请输入TG BOT Token(0返回)> ")
			token, err := readLine(reader)
			if err != nil {
				return err
			}
			token = strings.TrimSpace(token)
			if token == "" || token == "0" {
				continue
			}
			fmt.Fprint(out, "请输入TG Chat ID(0返回)> ")
			chatID, err := readLine(reader)
			if err != nil {
				return err
			}
			chatID = strings.TrimSpace(chatID)
			if chatID == "" || chatID == "0" {
				continue
			}
			if err := ConfigureTGBot(opts, token, chatID, out); err != nil {
				return err
			}
			fmt.Fprintln(out, "TG-BOT绑定设置完成")
		case "3":
			next, err := SetTGMenuPush(opts, "toggle")
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "启动时推送菜单: %s\n", next)
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunFirewallMenu(opts Options, in io.Reader, out io.Writer) error {
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
		fmt.Fprintln(out, "公网防火墙")
		fmt.Fprintf(out, "当前状态: %s\n", st.FWWan)
		if strings.TrimSpace(st.FWWanPorts) != "" {
			fmt.Fprintf(out, "手动放行端口: %s\n", st.FWWanPorts)
		}
		if strings.TrimSpace(st.VMSPort+st.SSSPort) != "" {
			fmt.Fprintf(out, "自动放行端口: %s %s\n", st.VMSPort, st.SSSPort)
		}
		fmt.Fprintf(out, "默认拦截端口: %d,%d\n", st.MixPort, st.DBPort)
		fmt.Fprintln(out, "1) 启用/关闭公网防火墙")
		fmt.Fprintln(out, "2) 添加放行端口")
		fmt.Fprintln(out, "3) 移除指定手动放行端口")
		fmt.Fprintln(out, "4) 清空全部手动放行端口")
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
			if err := toggleFirewall(opts, st, reader, out); err != nil {
				return err
			}
		case "2":
			if err := addFirewallPort(opts, st, reader, out); err != nil {
				return err
			}
		case "3":
			if err := removeFirewallPort(opts, st, reader, out); err != nil {
				return err
			}
		case "4":
			if err := setCfgValue(opts, "fw_wan_ports", ""); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunTrafficFilterMenu(opts Options, in io.Reader, out io.Writer) error {
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
		macPath := filepath.Join(opts.CrashDir, "configs", "mac")
		ipPath := filepath.Join(opts.CrashDir, "configs", "ip_filter")
		macEntries, _ := readListFile(macPath)
		ipEntries, _ := readListFile(ipPath)
		lanFilter := "OFF"
		if len(macEntries)+len(ipEntries) > 0 {
			lanFilter = "ON"
		}

		fmt.Fprintln(out, "流量过滤")
		fmt.Fprintf(out, "1) 过滤非常用端口: %s\n", st.CommonPorts)
		fmt.Fprintf(out, "2) 过滤局域网设备: %s\n", lanFilter)
		fmt.Fprintf(out, "3) 过滤QUIC协议: %s\n", st.QuicRJ)
		fmt.Fprintf(out, "4) 过滤CN_IP(4&6)列表: %s\n", st.CNIPRoute)
		fmt.Fprintln(out, "5) 自定义透明路由IPv4网段")
		fmt.Fprintln(out, "6) 自定义保留地址IPv4网段")
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
			if gatewayIsProcessRunning("CrashCore") && st.FirewallMod == "iptables" {
				fmt.Fprintln(out, "切换时将停止服务，是否继续？")
				fmt.Fprintln(out, "1) 是")
				fmt.Fprintln(out, "0) 否")
				fmt.Fprint(out, "请输入对应标号> ")
				confirm, err := readLine(reader)
				if err != nil {
					return err
				}
				if confirm != "1" {
					continue
				}
				if err := gatewayRunControllerAction(opts.CrashDir, "stop"); err != nil {
					return err
				}
			}
			if err := RunCommonPortsMenu(opts, reader, out); err != nil {
				return err
			}
		case "2":
			if err := RunLANDeviceFilterMenu(opts, reader, out); err != nil {
				return err
			}
		case "3":
			next := "ON"
			if strings.EqualFold(st.QuicRJ, "ON") {
				next = "OFF"
			}
			if err := setCfgValue(opts, "quic_rj", next); err != nil {
				return err
			}
			fmt.Fprintf(out, "quic_rj=%s\n", next)
		case "4":
			if !(gatewayHasIPSet() || st.FirewallMod == "nftables") {
				fmt.Fprintln(out, "当前设备缺少ipset模块或未使用nftables模式，无法启用绕过功能")
				continue
			}
			next := "ON"
			if strings.EqualFold(st.CNIPRoute, "ON") {
				next = "OFF"
			}
			if err := setCfgValue(opts, "cn_ip_route", next); err != nil {
				return err
			}
			fmt.Fprintf(out, "cn_ip_route=%s\n", next)
		case "5":
			if err := RunCustomHostIPv4Menu(opts, reader, out); err != nil {
				return err
			}
		case "6":
			if err := RunReserveIPv4Menu(opts, reader, out); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunLANDeviceFilterMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	macPath := filepath.Join(opts.CrashDir, "configs", "mac")
	ipPath := filepath.Join(opts.CrashDir, "configs", "ip_filter")
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		macEntries, err := readListFile(macPath)
		if err != nil {
			return err
		}
		ipEntries, err := readListFile(ipPath)
		if err != nil {
			return err
		}
		mode := strings.TrimSpace(st.MacFilterType)
		if mode == "" {
			mode = "黑名单"
		}
		alt := "白名单"
		neg := "不"
		if mode == "白名单" {
			alt = "黑名单"
			neg = ""
		}
		leases, _ := loadLeaseRecords()

		fmt.Fprintln(out, "局域网设备过滤")
		fmt.Fprintf(out, "当前过滤方式: %s模式\n", mode)
		fmt.Fprintf(out, "仅列表内设备流量%s经过内核\n", neg)
		if len(macEntries)+len(ipEntries) > 0 {
			fmt.Fprintln(out, "当前已过滤设备:")
			for _, dev := range append(append([]string{}, macEntries...), ipEntries...) {
				rec := lookupLeaseInfo(dev, leases)
				name := rec.Name
				if name == "" {
					name = "未知设备"
				}
				fmt.Fprintf(out, "- %s %s\n", dev, name)
			}
		}
		fmt.Fprintf(out, "1) 切换为%s模式\n", alt)
		fmt.Fprintln(out, "2) 添加指定设备(MAC地址)")
		fmt.Fprintln(out, "3) 添加指定设备(IP地址/网段)")
		fmt.Fprintln(out, "4) 移除指定设备")
		fmt.Fprintln(out, "9) 清空整个列表")
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
			if err := setCfgValue(opts, "macfilter_type", alt); err != nil {
				return err
			}
		case "2":
			next, err := runLANAddMACFlow(reader, out, macEntries, leases)
			if err != nil {
				return err
			}
			if next == nil {
				continue
			}
			if err := writeListFile(macPath, next); err != nil {
				return err
			}
		case "3":
			next, err := runLANAddIPFlow(reader, out, ipEntries, leases)
			if err != nil {
				return err
			}
			if next == nil {
				continue
			}
			if err := writeListFile(ipPath, next); err != nil {
				return err
			}
		case "4":
			changedMac, changedIP, changed, err := runLANRemoveFlow(reader, out, macEntries, ipEntries, leases)
			if err != nil {
				return err
			}
			if !changed {
				continue
			}
			if err := writeListFile(macPath, changedMac); err != nil {
				return err
			}
			if err := writeListFile(ipPath, changedIP); err != nil {
				return err
			}
		case "9":
			if err := writeListFile(macPath, nil); err != nil {
				return err
			}
			if err := writeListFile(ipPath, nil); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunCommonPortsMenu(opts Options, in io.Reader, out io.Writer) error {
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
		current := effectiveMultiport(st.Multiport)
		fmt.Fprintln(out, "非常用端口过滤")
		fmt.Fprintf(out, "过滤状态: %s\n", st.CommonPorts)
		fmt.Fprintf(out, "当前放行端口: %s\n", current)
		fmt.Fprintln(out, "1) 启用/关闭端口过滤")
		fmt.Fprintln(out, "2) 添加放行端口")
		fmt.Fprintln(out, "3) 移除指定放行端口")
		fmt.Fprintln(out, "4) 重置默认放行端口")
		fmt.Fprintln(out, "5) 重置为旧版放行端口")
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
			next := "ON"
			if st.CommonPorts == "ON" {
				next = "OFF"
			}
			if err := setCfgValue(opts, "common_ports", next); err != nil {
				return err
			}
		case "2":
			if err := addCommonPort(opts, current, reader, out); err != nil {
				return err
			}
		case "3":
			if err := removeCommonPort(opts, current, reader, out); err != nil {
				return err
			}
		case "4":
			if err := setCfgValue(opts, "multiport", ""); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		case "5":
			if err := setCfgValue(opts, "multiport", oldDefaultMultiport); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunCustomHostIPv4Menu(opts Options, in io.Reader, out io.Writer) error {
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
		hostIPv4 := ""
		if vars, err := firewall.GetHostVars(opts.CrashDir, nil); err == nil {
			hostIPv4 = strings.TrimSpace(vars.HostIPv4)
		}
		fmt.Fprintln(out, "自定义透明路由IPv4网段")
		fmt.Fprintf(out, "当前默认透明路由网段: %s\n", defaultText(hostIPv4, "未检测到"))
		fmt.Fprintf(out, "当前自定义网段: %s\n", defaultText(strings.TrimSpace(st.CustHostIPv4), "未设置"))
		fmt.Fprintf(out, "覆盖默认网段: %s\n", st.ReplaceHostV4)
		fmt.Fprintln(out, "1) 移除所有自定义网段")
		fmt.Fprintln(out, "2) 切换覆盖默认网段")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入序号或直接输入要添加的IPv4 CIDR网段> ")
		text, err := readLine(reader)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		switch text {
		case "", "0":
			return nil
		case "1":
			if err := setCfgValue(opts, "cust_host_ipv4", ""); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		case "2":
			next := "ON"
			if strings.EqualFold(st.ReplaceHostV4, "ON") {
				next = "OFF"
			}
			if err := setCfgValue(opts, "replace_default_host_ipv4", next); err != nil {
				return err
			}
			fmt.Fprintf(out, "replace_default_host_ipv4=%s\n", next)
		default:
			if _, _, err := net.ParseCIDR(text); err != nil {
				fmt.Fprintln(out, "请输入正确的IPv4网段地址")
				continue
			}
			cur := splitSpaceFields(st.CustHostIPv4)
			if containsString(cur, text) {
				fmt.Fprintln(out, "该网段已存在")
				continue
			}
			cur = append(cur, text)
			if err := setCfgValue(opts, "cust_host_ipv4", quote(strings.Join(cur, " "))); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		}
	}
}

func RunReserveIPv4Menu(opts Options, in io.Reader, out io.Writer) error {
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
		current := strings.TrimSpace(st.ReserveIPv4)
		if current == "" {
			current = defaultReserveIPv4
		}
		fmt.Fprintln(out, "自定义保留地址IPv4网段")
		fmt.Fprintln(out, "注意: 地址必须使用空格分隔的CIDR格式，设置错误会导致网络异常")
		fmt.Fprintf(out, "当前网段: %s\n", current)
		fmt.Fprintln(out, "1) 重置为默认网段")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入序号或直接输入保留地址IPv4网段> ")
		text, err := readLine(reader)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		switch text {
		case "", "0":
			return nil
		case "1":
			if err := setCfgValue(opts, "reserve_ipv4", ""); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		default:
			fields := splitSpaceFields(text)
			if len(fields) == 0 {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			valid := true
			for _, cidr := range fields {
				if _, _, err := net.ParseCIDR(cidr); err != nil {
					valid = false
					break
				}
			}
			if !valid {
				fmt.Fprintln(out, "输入有误，请输入空格分隔的IPv4 CIDR网段")
				continue
			}
			if err := setCfgValue(opts, "reserve_ipv4", quote(strings.Join(fields, " "))); err != nil {
				return err
			}
			fmt.Fprintln(out, "操作成功")
		}
	}
}

func toggleFirewall(opts Options, st State, reader *bufio.Reader, out io.Writer) error {
	next := "ON"
	if st.FWWan == "ON" {
		fmt.Fprintln(out, "是否确认关闭防火墙？这会带来极大的安全隐患！")
		fmt.Fprintln(out, "1) 是")
		fmt.Fprintln(out, "0) 否")
		fmt.Fprint(out, "请输入对应标号> ")
		res, err := readLine(reader)
		if err != nil {
			return err
		}
		if res != "1" {
			next = "ON"
		} else {
			next = "OFF"
		}
	}
	return setCfgValue(opts, "fw_wan", next)
}

func addFirewallPort(opts Options, st State, reader *bufio.Reader, out io.Writer) error {
	ports := parsePortList(st.FWWanPorts)
	if len(ports) >= 10 {
		fmt.Fprintln(out, "最多支持设置放行10个端口，请先减少一些")
		return nil
	}
	fmt.Fprint(out, "请输入要放行的端口号> ")
	raw, err := readLine(reader)
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 65535 {
		fmt.Fprintln(out, "输入错误，请输入正确的数值(1-65535)")
		return nil
	}
	p := strconv.Itoa(n)
	for _, v := range ports {
		if v == p {
			fmt.Fprintln(out, "输入错误，请勿重复添加")
			return nil
		}
	}
	ports = append(ports, p)
	return setCfgValue(opts, "fw_wan_ports", strings.Join(ports, ","))
}

const defaultMultiport = "22,80,443,8080,8443"
const oldDefaultMultiport = "22,80,143,194,443,465,587,853,993,995,5222,8080,8443"
const defaultReserveIPv4 = "0.0.0.0/8 10.0.0.0/8 127.0.0.0/8 100.64.0.0/10 169.254.0.0/16 172.16.0.0/12 192.168.0.0/16 224.0.0.0/4 240.0.0.0/4"

func addCommonPort(opts Options, current string, reader *bufio.Reader, out io.Writer) error {
	ports := parsePortList(current)
	if len(ports) >= 15 {
		fmt.Fprintln(out, "最多支持设置放行15个端口，请先减少一些")
		return nil
	}
	fmt.Fprint(out, "请输入要放行的端口号> ")
	raw, err := readLine(reader)
	if err != nil {
		return err
	}
	n, err := parsePort(raw)
	if err != nil {
		fmt.Fprintln(out, "输入错误，请输入正确的数值(1-65535)")
		return nil
	}
	p := strconv.Itoa(n)
	for _, v := range ports {
		if v == p {
			fmt.Fprintln(out, "输入错误，请勿重复添加")
			return nil
		}
	}
	ports = append(ports, p)
	return setCfgValue(opts, "multiport", strings.Join(ports, ","))
}

func removeCommonPort(opts Options, current string, reader *bufio.Reader, out io.Writer) error {
	ports := parsePortList(current)
	for {
		fmt.Fprintln(out, "请输入要移除的端口号，输入0返回")
		fmt.Fprint(out, "请输入> ")
		raw, err := readLine(reader)
		if err != nil {
			return err
		}
		if raw == "0" || raw == "" {
			return nil
		}
		n, err := parsePort(raw)
		if err != nil {
			fmt.Fprintln(out, "输入错误，请输入正确的数值(1-65535)")
			continue
		}
		p := strconv.Itoa(n)
		idx := -1
		for i, v := range ports {
			if v == p {
				idx = i
				break
			}
		}
		if idx < 0 {
			fmt.Fprintln(out, "输入错误，请输入已添加过的端口")
			continue
		}
		ports = append(ports[:idx], ports[idx+1:]...)
		return setCfgValue(opts, "multiport", strings.Join(ports, ","))
	}
}

func effectiveMultiport(raw string) string {
	raw = strings.TrimSpace(stripQuotes(raw))
	if raw == "" {
		return defaultMultiport
	}
	return raw
}

func removeFirewallPort(opts Options, st State, reader *bufio.Reader, out io.Writer) error {
	ports := parsePortList(st.FWWanPorts)
	if len(ports) == 0 {
		fmt.Fprintln(out, "当前没有已添加的端口")
		return nil
	}
	for {
		fmt.Fprintln(out, "请输入要移除的端口号，输入0返回")
		fmt.Fprint(out, "请输入> ")
		raw, err := readLine(reader)
		if err != nil {
			return err
		}
		if raw == "0" || raw == "" {
			return nil
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 65535 {
			fmt.Fprintln(out, "输入错误，请输入正确的数值(1-65535)")
			continue
		}
		p := strconv.Itoa(n)
		idx := -1
		for i, v := range ports {
			if v == p {
				idx = i
				break
			}
		}
		if idx < 0 {
			fmt.Fprintln(out, "输入错误，请输入已添加过的端口")
			continue
		}
		ports = append(ports[:idx], ports[idx+1:]...)
		return setCfgValue(opts, "fw_wan_ports", strings.Join(ports, ","))
	}
}

func RunVmessMenu(opts Options, in io.Reader, out io.Writer) error {
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
		fmt.Fprintln(out, "Vmess入站设置")
		fmt.Fprintf(out, "1) 启用/关闭: %s\n", st.VMSService)
		fmt.Fprintf(out, "2) 监听端口: %s\n", st.VMSPort)
		fmt.Fprintf(out, "3) WS-path: %s\n", st.VMSWSPath)
		fmt.Fprintf(out, "4) UUID: %s\n", st.VMSUUID)
		fmt.Fprintln(out, "5) 随机生成UUID")
		fmt.Fprintf(out, "6) 混淆host: %s\n", st.VMSHost)
		fmt.Fprintln(out, "7) 生成分享链接")
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
			if st.VMSService != "ON" {
				if strings.TrimSpace(st.VMSPort) == "" || strings.TrimSpace(st.VMSUUID) == "" {
					fmt.Fprintln(out, "请先完成必选设置")
					continue
				}
				next = "ON"
			}
			if err := setGatewayCfgValue(opts, "vms_service", next); err != nil {
				return err
			}
		case "2":
			fmt.Fprint(out, "请输入端口号（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				if err := setGatewayCfgValue(opts, "vms_port", ""); err != nil {
					return err
				}
				continue
			}
			if _, err := parsePort(raw); err != nil {
				fmt.Fprintln(out, "输入错误，请输入正确的数值(1-65535)")
				continue
			}
			if err := setGatewayCfgValue(opts, "vms_port", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "3":
			fmt.Fprint(out, "请输入ws-path路径（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				if err := setGatewayCfgValue(opts, "vms_ws_path", ""); err != nil {
					return err
				}
				continue
			}
			if !strings.HasPrefix(raw, "/") {
				fmt.Fprintln(out, "不是合法的path路径，必须以/开头")
				continue
			}
			if err := setGatewayCfgValue(opts, "vms_ws_path", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "4":
			fmt.Fprint(out, "请输入UUID（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				if err := setGatewayCfgValue(opts, "vms_uuid", ""); err != nil {
					return err
				}
				continue
			}
			if !vmessUUIDPattern.MatchString(strings.TrimSpace(raw)) {
				fmt.Fprintln(out, "不是合法的UUID格式")
				continue
			}
			if err := setGatewayCfgValue(opts, "vms_uuid", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "5":
			uuid, err := randomUUIDv4()
			if err != nil {
				return err
			}
			if err := setGatewayCfgValue(opts, "vms_uuid", uuid); err != nil {
				return err
			}
			fmt.Fprintf(out, "已生成UUID: %s\n", uuid)
		case "6":
			fmt.Fprint(out, "请输入免流混淆host（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				if err := setGatewayCfgValue(opts, "vms_host", ""); err != nil {
					return err
				}
				continue
			}
			if err := setGatewayCfgValue(opts, "vms_host", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "7":
			fmt.Fprint(out, "请输入本机公网IP(4/6)或域名> ")
			host, err := readLine(reader)
			if err != nil {
				return err
			}
			host = strings.TrimSpace(host)
			if host == "" || strings.TrimSpace(st.VMSPort) == "" || strings.TrimSpace(st.VMSUUID) == "" {
				fmt.Fprintln(out, "请先完成必选设置")
				continue
			}
			vmsNet := ""
			if strings.TrimSpace(st.VMSWSPath) != "" {
				vmsNet = "ws"
			}
			payload := map[string]string{
				"v":    "2",
				"ps":   "ShellCrash_vms_in",
				"add":  host,
				"port": st.VMSPort,
				"id":   st.VMSUUID,
				"aid":  "0",
				"type": "auto",
				"net":  vmsNet,
				"path": st.VMSWSPath,
				"host": st.VMSHost,
			}
			rawJSON, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			link := "vmess://" + base64.StdEncoding.EncodeToString(rawJSON)
			fmt.Fprintf(out, "%s\n", link)
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunShadowsocksMenu(opts Options, in io.Reader, out io.Writer) error {
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
		pwdDisplay := st.SSSPwd
		if strings.TrimSpace(pwdDisplay) == "" {
			pwdDisplay = "未设置"
		}
		fmt.Fprintln(out, "ShadowSocks入站设置")
		fmt.Fprintf(out, "1) 启用/关闭: %s\n", st.SSSService)
		fmt.Fprintf(out, "2) 监听端口: %s\n", st.SSSPort)
		fmt.Fprintf(out, "3) 加密协议: %s\n", st.SSSCipher)
		fmt.Fprintf(out, "4) Password: %s\n", pwdDisplay)
		fmt.Fprintln(out, "5) 生成分享链接")
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
			if st.SSSService != "ON" {
				if strings.TrimSpace(st.SSSPort) == "" || strings.TrimSpace(st.SSSCipher) == "" || strings.TrimSpace(st.SSSPwd) == "" {
					fmt.Fprintln(out, "请先完成必选设置")
					continue
				}
				next = "ON"
			}
			if err := setGatewayCfgValue(opts, "sss_service", next); err != nil {
				return err
			}
		case "2":
			fmt.Fprint(out, "请输入端口号（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				if err := setGatewayCfgValue(opts, "sss_port", ""); err != nil {
					return err
				}
				continue
			}
			if _, err := parsePort(raw); err != nil {
				fmt.Fprintln(out, "输入错误，请输入正确的数值(1-65535)")
				continue
			}
			if err := setGatewayCfgValue(opts, "sss_port", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "3":
			cipher, pwd, ok, err := chooseCipher(reader, out)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			if err := setKVValues(filepath.Join(opts.CrashDir, "configs", "gateway.cfg"), map[string]string{
				"sss_cipher": cipher,
				"sss_pwd":    pwd,
			}); err != nil {
				return err
			}
		case "4":
			if strings.Contains(st.SSSCipher, "2022-blake3") {
				fmt.Fprintln(out, "2022系列加密必须使用脚本随机生成的password")
				continue
			}
			fmt.Fprint(out, "请输入秘钥（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				if err := setGatewayCfgValue(opts, "sss_pwd", ""); err != nil {
					return err
				}
				continue
			}
			if err := setGatewayCfgValue(opts, "sss_pwd", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "5":
			fmt.Fprint(out, "请输入本机公网IP(4/6)或域名> ")
			host, err := readLine(reader)
			if err != nil {
				return err
			}
			host = strings.TrimSpace(host)
			if host == "" || strings.TrimSpace(st.SSSPort) == "" || strings.TrimSpace(st.SSSCipher) == "" || strings.TrimSpace(st.SSSPwd) == "" {
				fmt.Fprintln(out, "请先完成必选设置")
				continue
			}
			ssAuth := base64.StdEncoding.EncodeToString([]byte(st.SSSCipher + ":" + st.SSSPwd))
			link := "ss://" + ssAuth + "@" + host + ":" + st.SSSPort + "#ShellCrash_ss_in"
			fmt.Fprintf(out, "%s\n", link)
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunTailscaleMenu(opts Options, in io.Reader, out io.Writer) error {
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
		keyInfo := ""
		if strings.TrimSpace(st.TSAuthKey) != "" {
			keyInfo = "*********"
		}
		fmt.Fprintln(out, "Tailscale设置")
		fmt.Fprintf(out, "1) 启用/关闭服务: %s\n", st.TSService)
		fmt.Fprintf(out, "2) 秘钥(Auth Key): %s\n", keyInfo)
		fmt.Fprintf(out, "3) 通告内网地址(Subnet): %t\n", st.TSSubnet)
		fmt.Fprintf(out, "4) 通告全部流量(EXIT-NODE): %t\n", st.TSExitNode)
		fmt.Fprintf(out, "5) 设备名称: %s\n", st.TSHostname)
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
			if strings.TrimSpace(st.TSAuthKey) == "" {
				fmt.Fprintln(out, "请先设置秘钥")
				continue
			}
			next := "ON"
			if st.TSService == "ON" {
				next = "OFF"
			}
			if err := setGatewayCfgValue(opts, "ts_service", next); err != nil {
				return err
			}
		case "2":
			fmt.Fprint(out, "请输入秘钥（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				raw = ""
			}
			if err := setGatewayCfgValue(opts, "ts_auth_key", strings.TrimSpace(raw)); err != nil {
				return err
			}
		case "3":
			next := "true"
			if st.TSSubnet {
				next = "false"
			}
			if err := setGatewayCfgValue(opts, "ts_subnet", next); err != nil {
				return err
			}
		case "4":
			next := "true"
			if st.TSExitNode {
				next = "false"
			}
			if err := setGatewayCfgValue(opts, "ts_exit_node", next); err != nil {
				return err
			}
		case "5":
			fmt.Fprint(out, "请输入设备名称（输入0返回）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			if strings.TrimSpace(raw) == "0" {
				continue
			}
			if err := setGatewayCfgValue(opts, "ts_hostname", strings.TrimSpace(raw)); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunWireGuardMenu(opts Options, in io.Reader, out io.Writer) error {
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
		pubInfo := ""
		if strings.TrimSpace(st.WGPublicKey) != "" {
			pubInfo = "*********"
		}
		pskInfo := ""
		if strings.TrimSpace(st.WGPreSharedKey) != "" {
			pskInfo = "*********"
		}
		priInfo := ""
		if strings.TrimSpace(st.WGPrivateKey) != "" {
			priInfo = "*********"
		}
		fmt.Fprintln(out, "WireGuard设置")
		fmt.Fprintf(out, "1) 启用/关闭服务: %s\n", st.WGService)
		fmt.Fprintf(out, "2) Endpoint地址: %s\n", st.WGServer)
		fmt.Fprintf(out, "3) Endpoint端口: %s\n", st.WGPort)
		fmt.Fprintf(out, "4) PublicKey: %s\n", pubInfo)
		fmt.Fprintf(out, "5) PresharedKey: %s\n", pskInfo)
		fmt.Fprintf(out, "6) PrivateKey: %s\n", priInfo)
		fmt.Fprintf(out, "7) 组网IPV4: %s\n", st.WGIPv4)
		fmt.Fprintf(out, "8) 组网IPV6: %s\n", st.WGIPv6)
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
			if !wireGuardRequiredReady(st) {
				fmt.Fprintln(out, "请先完成必选设置")
				continue
			}
			next := "ON"
			if st.WGService == "ON" {
				next = "OFF"
			}
			if err := setGatewayCfgValue(opts, "wg_service", next); err != nil {
				return err
			}
		case "2", "3", "4", "5", "6", "7", "8":
			fmt.Fprint(out, "请输入相应内容（输入0删除）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			value := strings.TrimSpace(raw)
			if value == "0" {
				value = ""
			}
			key := ""
			switch choice {
			case "2":
				key = "wg_server"
			case "3":
				key = "wg_port"
			case "4":
				key = "wg_public_key"
			case "5":
				key = "wg_pre_shared_key"
			case "6":
				key = "wg_private_key"
			case "7":
				key = "wg_ipv4"
			case "8":
				key = "wg_ipv6"
			}
			if err := setGatewayCfgValue(opts, key, value); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func wireGuardRequiredReady(st State) bool {
	return strings.TrimSpace(st.WGServer) != "" &&
		strings.TrimSpace(st.WGPort) != "" &&
		strings.TrimSpace(st.WGPublicKey) != "" &&
		strings.TrimSpace(st.WGPreSharedKey) != "" &&
		strings.TrimSpace(st.WGPrivateKey) != "" &&
		strings.TrimSpace(st.WGIPv4) != ""
}

func LoadState(opts Options) (State, error) {
	opts = withDefaults(opts)
	cfgPath := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	gwPath := filepath.Join(opts.CrashDir, "configs", "gateway.cfg")
	cfgKV, err := parseKVFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return State{}, err
	}
	gwKV, err := parseKVFile(gwPath)
	if err != nil && !os.IsNotExist(err) {
		return State{}, err
	}
	if cfgKV == nil {
		cfgKV = map[string]string{}
	}
	if gwKV == nil {
		gwKV = map[string]string{}
	}
	st := State{}
	st.CrashCore = strings.TrimSpace(stripQuotes(cfgKV["crashcore"]))
	st.FirewallMod = strings.TrimSpace(stripQuotes(cfgKV["firewall_mod"]))
	if st.CrashCore == "" {
		st.CrashCore = "meta"
	}
	if st.FirewallMod == "" {
		st.FirewallMod = "iptables"
	}
	st.FWWan = defaultONOFF(firstNonEmpty(gwKV["fw_wan"], cfgKV["fw_wan"]), "ON")
	st.FWWanPorts = stripQuotes(firstNonEmpty(gwKV["fw_wan_ports"], cfgKV["fw_wan_ports"]))
	st.CommonPorts = defaultONOFF(cfgKV["common_ports"], "ON")
	st.QuicRJ = defaultONOFF(cfgKV["quic_rj"], "OFF")
	st.CNIPRoute = defaultONOFF(cfgKV["cn_ip_route"], "OFF")
	st.MacFilterType = defaultString(stripQuotes(cfgKV["macfilter_type"]), "黑名单")
	st.CustHostIPv4 = stripQuotes(cfgKV["cust_host_ipv4"])
	st.ReplaceHostV4 = defaultONOFF(cfgKV["replace_default_host_ipv4"], "OFF")
	st.ReserveIPv4 = stripQuotes(cfgKV["reserve_ipv4"])
	st.Multiport = stripQuotes(cfgKV["multiport"])
	st.MixPort = defaultInt(cfgKV["mix_port"], 7890)
	st.DBPort = defaultInt(cfgKV["db_port"], 9999)
	st.TSService = defaultONOFF(firstNonEmpty(gwKV["ts_service"], cfgKV["ts_service"]), "OFF")
	st.TSAuthKey = stripQuotes(firstNonEmpty(gwKV["ts_auth_key"], cfgKV["ts_auth_key"]))
	st.TSSubnet = defaultBool(firstNonEmpty(gwKV["ts_subnet"], cfgKV["ts_subnet"]), false)
	st.TSExitNode = defaultBool(firstNonEmpty(gwKV["ts_exit_node"], cfgKV["ts_exit_node"]), false)
	st.TSHostname = stripQuotes(firstNonEmpty(gwKV["ts_hostname"], cfgKV["ts_hostname"]))
	st.WGService = defaultONOFF(firstNonEmpty(gwKV["wg_service"], cfgKV["wg_service"]), "OFF")
	st.WGServer = stripQuotes(firstNonEmpty(gwKV["wg_server"], cfgKV["wg_server"]))
	st.WGPort = stripQuotes(firstNonEmpty(gwKV["wg_port"], cfgKV["wg_port"]))
	st.WGPublicKey = stripQuotes(firstNonEmpty(gwKV["wg_public_key"], cfgKV["wg_public_key"]))
	st.WGPreSharedKey = stripQuotes(firstNonEmpty(gwKV["wg_pre_shared_key"], cfgKV["wg_pre_shared_key"]))
	st.WGPrivateKey = stripQuotes(firstNonEmpty(gwKV["wg_private_key"], cfgKV["wg_private_key"]))
	st.WGIPv4 = stripQuotes(firstNonEmpty(gwKV["wg_ipv4"], cfgKV["wg_ipv4"]))
	st.WGIPv6 = stripQuotes(firstNonEmpty(gwKV["wg_ipv6"], cfgKV["wg_ipv6"]))
	st.VMSPort = stripQuotes(firstNonEmpty(gwKV["vms_port"], cfgKV["vms_port"]))
	st.SSSPort = stripQuotes(firstNonEmpty(gwKV["sss_port"], cfgKV["sss_port"]))
	st.VMSService = defaultONOFF(firstNonEmpty(gwKV["vms_service"], cfgKV["vms_service"]), "OFF")
	st.VMSWSPath = stripQuotes(firstNonEmpty(gwKV["vms_ws_path"], cfgKV["vms_ws_path"]))
	st.VMSUUID = stripQuotes(firstNonEmpty(gwKV["vms_uuid"], cfgKV["vms_uuid"]))
	st.VMSHost = stripQuotes(firstNonEmpty(gwKV["vms_host"], cfgKV["vms_host"]))
	st.SSSService = defaultONOFF(firstNonEmpty(gwKV["sss_service"], cfgKV["sss_service"]), "OFF")
	st.SSSCipher = stripQuotes(firstNonEmpty(gwKV["sss_cipher"], cfgKV["sss_cipher"]))
	st.SSSPwd = stripQuotes(firstNonEmpty(gwKV["sss_pwd"], cfgKV["sss_pwd"]))
	st.BotTGService = defaultONOFF(firstNonEmpty(gwKV["bot_tg_service"], cfgKV["bot_tg_service"]), "OFF")
	st.TGMenuPush = defaultONOFF(firstNonEmpty(gwKV["TG_menupush"], cfgKV["TG_menupush"]), "OFF")
	st.TGChatID = stripQuotes(firstNonEmpty(gwKV["TG_CHATID"], cfgKV["TG_CHATID"]))
	return st, nil
}

func ConfigureTGBot(opts Options, token, chatID string, out io.Writer) error {
	opts = withDefaults(opts)
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)
	if token == "" || chatID == "" {
		return fmt.Errorf("token/chat-id cannot be empty")
	}
	path := filepath.Join(opts.CrashDir, "configs", "gateway.cfg")
	if err := setKVValues(path, map[string]string{
		"TG_TOKEN":  token,
		"TG_CHATID": chatID,
	}); err != nil {
		return err
	}

	alias := "crash"
	mainCfgPath := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	if mainKV, err := parseKVFile(mainCfgPath); err == nil {
		if v := strings.TrimSpace(stripQuotes(mainKV["my_alias"])); v != "" {
			alias = v
		}
	}
	apiRoot := "https://api.telegram.org/bot" + token
	commandsPayload := map[string]any{
		"commands": []map[string]string{
			{"command": alias, "description": "呼出ShellCrash菜单"},
			{"command": "help", "description": "查看帮助"},
		},
	}
	textPayload := map[string]string{
		"chat_id": chatID,
		"text":    "已完成Telegram机器人设置！请使用 /" + alias + " 呼出功能菜单！",
	}
	if err := postTelegramJSON(apiRoot+"/setMyCommands", commandsPayload); err != nil && out != nil {
		fmt.Fprintf(out, "warning: setMyCommands failed: %v\n", err)
	}
	if err := postTelegramJSON(apiRoot+"/sendMessage", textPayload); err != nil && out != nil {
		fmt.Fprintf(out, "warning: sendMessage failed: %v\n", err)
	}
	return nil
}

func SetTGService(opts Options, mode string) (string, error) {
	opts = withDefaults(opts)
	gwPath := filepath.Join(opts.CrashDir, "configs", "gateway.cfg")
	kv, err := parseKVFile(gwPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	current := defaultONOFF(kv["bot_tg_service"], "OFF")
	next, err := resolveMode(mode, current)
	if err != nil {
		return "", err
	}
	if next == "OFF" {
		if err := gatewayRunControllerAction(opts.CrashDir, "bot_tg_stop"); err != nil {
			return "", err
		}
	} else if gatewayIsProcessRunning("CrashCore") {
		if err := gatewayRunControllerAction(opts.CrashDir, "bot_tg_start"); err != nil {
			return "", err
		}
		if err := gatewayRunControllerAction(opts.CrashDir, "bot_tg_cron"); err != nil {
			return "", err
		}
	}
	if err := setKVValues(gwPath, map[string]string{"bot_tg_service": next}); err != nil {
		return "", err
	}
	return next, nil
}

func SetTGMenuPush(opts Options, mode string) (string, error) {
	opts = withDefaults(opts)
	gwPath := filepath.Join(opts.CrashDir, "configs", "gateway.cfg")
	kv, err := parseKVFile(gwPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	current := defaultONOFF(kv["TG_menupush"], "OFF")
	next, err := resolveMode(mode, current)
	if err != nil {
		return "", err
	}
	if err := setKVValues(gwPath, map[string]string{"TG_menupush": next}); err != nil {
		return "", err
	}
	return next, nil
}

func withDefaults(opts Options) Options {
	if strings.TrimSpace(opts.CrashDir) == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	return opts
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

func setGatewayCfgValue(opts Options, key, value string) error {
	path := filepath.Join(opts.CrashDir, "configs", "gateway.cfg")
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

func setKVValues(path string, values map[string]string) error {
	kv, err := parseKVFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	for k, v := range values {
		if strings.TrimSpace(v) == "" {
			delete(kv, k)
			continue
		}
		kv[k] = v
	}
	return writeKVFile(path, kv)
}

func resolveMode(mode, current string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "toggle":
		if strings.EqualFold(current, "ON") {
			return "OFF", nil
		}
		return "ON", nil
	case "on":
		return "ON", nil
	case "off":
		return "OFF", nil
	default:
		return "", fmt.Errorf("unsupported mode %q", mode)
	}
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return "'" + s + "'"
}

func splitSpaceFields(raw string) []string {
	raw = strings.TrimSpace(stripQuotes(raw))
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func defaultText(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultString(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
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

func defaultInt(v string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(stripQuotes(v)))
	if err != nil {
		return fallback
	}
	return n
}

func defaultBool(v string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(stripQuotes(v))) {
	case "1", "true", "on", "yes":
		return true
	case "0", "false", "off", "no":
		return false
	default:
		return fallback
	}
}

func parsePortList(raw string) []string {
	raw = strings.TrimSpace(stripQuotes(raw))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

type leaseRecord struct {
	MAC  string
	IP   string
	Name string
}

func runLANAddMACFlow(reader *bufio.Reader, out io.Writer, entries []string, leases []leaseRecord) ([]string, error) {
	fmt.Fprintln(out, "已添加的MAC地址:")
	if len(entries) == 0 {
		fmt.Fprintln(out, "暂无")
	} else {
		for _, e := range entries {
			fmt.Fprintln(out, e)
		}
	}
	if len(leases) > 0 {
		fmt.Fprintln(out, "可选设备:")
		for i, rec := range leases {
			fmt.Fprintf(out, "%d) %s %s %s\n", i+1, rec.IP, rec.MAC, rec.Name)
		}
	}
	fmt.Fprint(out, "请输入序号或MAC地址(0返回)> ")
	raw, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return nil, nil
	}
	val := ""
	if macPattern.MatchString(raw) {
		val = strings.ToLower(raw)
	} else if idx, err := strconv.Atoi(raw); err == nil && idx >= 1 && idx <= len(leases) {
		val = strings.ToLower(strings.TrimSpace(leases[idx-1].MAC))
	}
	if val == "" {
		fmt.Fprintln(out, "输入有误，请重新输入")
		return nil, nil
	}
	if containsString(entries, val) {
		fmt.Fprintln(out, "已添加的设备，请勿重复添加")
		return nil, nil
	}
	return append(entries, val), nil
}

func runLANAddIPFlow(reader *bufio.Reader, out io.Writer, entries []string, leases []leaseRecord) ([]string, error) {
	fmt.Fprintln(out, "已添加的IP地址(段):")
	if len(entries) == 0 {
		fmt.Fprintln(out, "暂无")
	} else {
		for _, e := range entries {
			fmt.Fprintln(out, e)
		}
	}
	if len(leases) > 0 {
		fmt.Fprintln(out, "可选设备:")
		for i, rec := range leases {
			fmt.Fprintf(out, "%d) %s %s\n", i+1, rec.IP, rec.Name)
		}
	}
	fmt.Fprint(out, "请输入序号或IP地址段(0返回)> ")
	raw, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return nil, nil
	}
	val := ""
	if isIPv4OrCIDR(raw) {
		val = raw
	} else if idx, err := strconv.Atoi(raw); err == nil && idx >= 1 && idx <= len(leases) {
		val = strings.TrimSpace(leases[idx-1].IP)
	}
	if val == "" {
		fmt.Fprintln(out, "输入有误，请重新输入")
		return nil, nil
	}
	if containsString(entries, val) {
		fmt.Fprintln(out, "已添加的地址，请勿重复添加")
		return nil, nil
	}
	return append(entries, val), nil
}

func runLANRemoveFlow(reader *bufio.Reader, out io.Writer, macEntries, ipEntries []string, leases []leaseRecord) ([]string, []string, bool, error) {
	combined := append(append([]string{}, macEntries...), ipEntries...)
	if len(combined) == 0 {
		fmt.Fprintln(out, "列表中没有需要移除的设备")
		return macEntries, ipEntries, false, nil
	}
	fmt.Fprintln(out, "请选择需要移除的设备:")
	for i, item := range combined {
		rec := lookupLeaseInfo(item, leases)
		fmt.Fprintf(out, "%d) %s %s\n", i+1, item, defaultText(rec.Name, "未知设备"))
	}
	fmt.Fprint(out, "请输入序号(0返回)> ")
	raw, err := readLine(reader)
	if err != nil {
		return macEntries, ipEntries, false, err
	}
	if raw == "" || raw == "0" {
		return macEntries, ipEntries, false, nil
	}
	idx, err := strconv.Atoi(raw)
	if err != nil || idx < 1 || idx > len(combined) {
		fmt.Fprintln(out, "输入有误，请重新输入")
		return macEntries, ipEntries, false, nil
	}
	if idx <= len(macEntries) {
		pos := idx - 1
		return append(macEntries[:pos], macEntries[pos+1:]...), ipEntries, true, nil
	}
	pos := idx - 1 - len(macEntries)
	return macEntries, append(ipEntries[:pos], ipEntries[pos+1:]...), true, nil
}

func isIPv4OrCIDR(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if ip := net.ParseIP(raw); ip != nil {
		return ip.To4() != nil
	}
	ip, ipNet, err := net.ParseCIDR(raw)
	if err != nil || ip == nil || ipNet == nil || ip.To4() == nil {
		return false
	}
	bits, _ := ipNet.Mask.Size()
	return bits >= 0 && bits <= 32
}

func readListFile(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

func writeListFile(path string, items []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(items) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	return os.WriteFile(path, []byte(strings.Join(items, "\n")+"\n"), 0o644)
}

func loadLeaseRecords() ([]leaseRecord, error) {
	paths := []string{
		"/var/lib/dhcp/dhcpd.leases",
		"/var/lib/dhcpd/dhcpd.leases",
		"/tmp/dhcp.leases",
		"/tmp/dnsmasq.leases",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return parseLeaseRecords(string(data)), nil
	}
	return nil, nil
}

func parseLeaseRecords(raw string) []leaseRecord {
	lines := strings.Split(raw, "\n")
	out := make([]leaseRecord, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		mac, ip, name := "", "", ""
		if len(fields) >= 4 && macPattern.MatchString(fields[1]) && net.ParseIP(fields[2]) != nil {
			mac = strings.ToLower(fields[1])
			ip = fields[2]
			name = fields[3]
		} else if macPattern.MatchString(fields[0]) && net.ParseIP(fields[1]) != nil {
			mac = strings.ToLower(fields[0])
			ip = fields[1]
			if len(fields) > 2 {
				name = fields[2]
			}
		} else {
			continue
		}
		if name == "*" {
			name = ""
		}
		out = append(out, leaseRecord{MAC: mac, IP: ip, Name: name})
	}
	return out
}

func lookupLeaseInfo(token string, leases []leaseRecord) leaseRecord {
	token = strings.TrimSpace(strings.ToLower(token))
	for _, rec := range leases {
		if strings.EqualFold(rec.MAC, token) || strings.EqualFold(rec.IP, token) {
			return rec
		}
	}
	return leaseRecord{}
}

func parsePort(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("invalid port")
	}
	return n, nil
}

func chooseCipher(reader *bufio.Reader, out io.Writer) (cipher string, password string, ok bool, err error) {
	fmt.Fprintln(out, "请选择加密协议:")
	fmt.Fprintln(out, "1) xchacha20-ietf-poly1305")
	fmt.Fprintln(out, "2) chacha20-ietf-poly1305")
	fmt.Fprintln(out, "3) aes-128-gcm")
	fmt.Fprintln(out, "4) aes-256-gcm")
	fmt.Fprintln(out, "5) 2022-blake3-chacha20-poly1305")
	fmt.Fprintln(out, "6) 2022-blake3-aes-128-gcm")
	fmt.Fprintln(out, "7) 2022-blake3-aes-256-gcm")
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应标号> ")
	num, err := readLine(reader)
	if err != nil {
		return "", "", false, err
	}
	switch strings.TrimSpace(num) {
	case "", "0":
		return "", "", false, nil
	case "1":
		pwd, err := randomBase64Bytes(16)
		return "xchacha20-ietf-poly1305", pwd, err == nil, err
	case "2":
		pwd, err := randomBase64Bytes(16)
		return "chacha20-ietf-poly1305", pwd, err == nil, err
	case "3":
		pwd, err := randomBase64Bytes(16)
		return "aes-128-gcm", pwd, err == nil, err
	case "4":
		pwd, err := randomBase64Bytes(16)
		return "aes-256-gcm", pwd, err == nil, err
	case "5":
		pwd, err := randomBase64Bytes(32)
		return "2022-blake3-chacha20-poly1305", pwd, err == nil, err
	case "6":
		pwd, err := randomBase64Bytes(16)
		return "2022-blake3-aes-128-gcm", pwd, err == nil, err
	case "7":
		pwd, err := randomBase64Bytes(32)
		return "2022-blake3-aes-256-gcm", pwd, err == nil, err
	default:
		fmt.Fprintln(out, "输入错误")
		return "", "", false, nil
	}
}

func randomBase64Bytes(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func randomUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}

func postTelegramJSON(url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := gatewayHTTPDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
