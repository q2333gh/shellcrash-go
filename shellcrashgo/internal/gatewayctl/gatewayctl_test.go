package gatewayctl

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFirewallMenuToggleAndAddPort(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("fw_wan=ON\nmix_port=7890\ndb_port=9999\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("1\n1\n2\n12345\n0\n")
	var out bytes.Buffer
	if err := RunFirewallMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if !strings.Contains(text, "fw_wan=OFF") {
		t.Fatalf("expected fw_wan=OFF, got: %s", text)
	}
	if !strings.Contains(text, "fw_wan_ports=12345") {
		t.Fatalf("expected fw_wan_ports set, got: %s", text)
	}
}

func TestRunFirewallMenuRemoveAndClearPort(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	cfgText := "fw_wan=ON\nfw_wan_ports=1000,2000\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("3\n2000\n4\n0\n")
	var out bytes.Buffer
	if err := RunFirewallMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if strings.Contains(text, "fw_wan_ports=") {
		t.Fatalf("expected fw_wan_ports cleared, got: %s", text)
	}
}

func TestRunCommonPortsMenuToggleAndAddPort(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("common_ports=ON\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("1\n2\n12345\n0\n")
	var out bytes.Buffer
	if err := RunCommonPortsMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if !strings.Contains(text, "common_ports=OFF") {
		t.Fatalf("expected common_ports=OFF, got: %s", text)
	}
	if !strings.Contains(text, "multiport=22,80,443,8080,8443,12345") {
		t.Fatalf("expected updated multiport, got: %s", text)
	}
}

func TestRunCommonPortsMenuRemoveAndReset(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	cfgText := "common_ports=ON\nmultiport=22,80,443,8080,8443,12345\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("3\n12345\n5\n4\n0\n")
	var out bytes.Buffer
	if err := RunCommonPortsMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if strings.Contains(text, "multiport=") {
		t.Fatalf("expected multiport cleared after reset, got: %s", text)
	}
}

func TestRunCustomHostIPv4MenuAddToggleAndClear(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	in := strings.NewReader("192.168.10.0/24\n2\n1\n0\n")
	var out bytes.Buffer
	if err := RunCustomHostIPv4Menu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if strings.Contains(text, "cust_host_ipv4=") {
		t.Fatalf("expected cust_host_ipv4 cleared, got: %s", text)
	}
	if !strings.Contains(text, "replace_default_host_ipv4=ON") {
		t.Fatalf("expected replace_default_host_ipv4=ON, got: %s", text)
	}
}

func TestRunReserveIPv4MenuSetAndReset(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	in := strings.NewReader("10.0.0.0/8 192.168.0.0/16\n1\n0\n")
	var out bytes.Buffer
	if err := RunReserveIPv4Menu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if strings.Contains(text, "reserve_ipv4=") {
		t.Fatalf("expected reserve_ipv4 cleared after reset, got: %s", text)
	}
}

func TestRunTrafficFilterMenuTogglesQUICAndCNIP(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("quic_rj=OFF\ncn_ip_route=OFF\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origHasIPSet := gatewayHasIPSet
	defer func() { gatewayHasIPSet = origHasIPSet }()
	gatewayHasIPSet = func() bool { return true }

	in := strings.NewReader("3\n4\n0\n")
	var out bytes.Buffer
	if err := RunTrafficFilterMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if !strings.Contains(text, "quic_rj=ON") {
		t.Fatalf("expected quic_rj=ON, got: %s", text)
	}
	if !strings.Contains(text, "cn_ip_route=ON") {
		t.Fatalf("expected cn_ip_route=ON, got: %s", text)
	}
}

func TestRunLANDeviceFilterMenuAddToggleAndRemove(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	in := strings.NewReader("1\n2\naa:bb:cc:dd:ee:ff\n3\n192.168.1.20\n4\n1\n0\n")
	var out bytes.Buffer
	if err := RunLANDeviceFilterMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if !strings.Contains(string(cfgOut), "macfilter_type=白名单") {
		t.Fatalf("expected macfilter_type switched, got: %s", string(cfgOut))
	}
	macOut, err := os.ReadFile(filepath.Join(cfgDir, "mac"))
	if err != nil {
		t.Fatalf("read mac list: %v", err)
	}
	if strings.TrimSpace(string(macOut)) != "" {
		t.Fatalf("expected mac entry removed, got: %q", string(macOut))
	}
	ipOut, err := os.ReadFile(filepath.Join(cfgDir, "ip_filter"))
	if err != nil {
		t.Fatalf("read ip list: %v", err)
	}
	if strings.TrimSpace(string(ipOut)) != "192.168.1.20" {
		t.Fatalf("unexpected ip filter content: %q", string(ipOut))
	}
}

func TestRunGatewayMenuStopsServiceBeforeFirewallOnIptables(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	cfgText := "firewall_mod=iptables\ncrashcore=meta\nfw_wan=ON\nmix_port=7890\ndb_port=9999\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	oldRun := gatewayRunControllerAction
	oldRunning := gatewayIsProcessRunning
	defer func() {
		gatewayRunControllerAction = oldRun
		gatewayIsProcessRunning = oldRunning
	}()
	var calls []string
	gatewayRunControllerAction = func(_ string, action string) error {
		calls = append(calls, action)
		return nil
	}
	gatewayIsProcessRunning = func(name string) bool { return name == "CrashCore" }

	input := "1\n1\n0\n0\n"
	var out bytes.Buffer
	if err := RunGatewayMenu(Options{CrashDir: crashDir}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run gateway menu: %v", err)
	}
	if len(calls) != 1 || calls[0] != "stop" {
		t.Fatalf("expected stop action before fw menu, got: %#v", calls)
	}
}

func TestRunGatewayMenuDispatchesDDNSMenu(t *testing.T) {
	oldDDNS := gatewayRunDDNSMenu
	defer func() { gatewayRunDDNSMenu = oldDDNS }()
	called := false
	gatewayRunDDNSMenu = func(in io.Reader, out io.Writer) error {
		called = true
		return nil
	}

	input := "3\n0\n"
	var out bytes.Buffer
	if err := RunGatewayMenu(Options{CrashDir: t.TempDir()}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run gateway menu: %v", err)
	}
	if !called {
		t.Fatalf("expected ddns menu dispatch")
	}
}

func TestSetTGServiceToggle(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "gateway.cfg"), []byte("bot_tg_service=OFF\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	oldRun := gatewayRunControllerAction
	oldRunning := gatewayIsProcessRunning
	defer func() {
		gatewayRunControllerAction = oldRun
		gatewayIsProcessRunning = oldRunning
	}()
	var calls []string
	gatewayRunControllerAction = func(_ string, action string) error {
		calls = append(calls, action)
		return nil
	}
	gatewayIsProcessRunning = func(name string) bool { return name == "CrashCore" }

	next, err := SetTGService(Options{CrashDir: crashDir}, "toggle")
	if err != nil {
		t.Fatalf("toggle on: %v", err)
	}
	if next != "ON" {
		t.Fatalf("expected ON, got %s", next)
	}
	if got := strings.Join(calls, ","); got != "bot_tg_start,bot_tg_cron" {
		t.Fatalf("unexpected actions: %s", got)
	}
	calls = nil
	next, err = SetTGService(Options{CrashDir: crashDir}, "toggle")
	if err != nil {
		t.Fatalf("toggle off: %v", err)
	}
	if next != "OFF" {
		t.Fatalf("expected OFF, got %s", next)
	}
	if got := strings.Join(calls, ","); got != "bot_tg_stop" {
		t.Fatalf("unexpected actions: %s", got)
	}
}

func TestSetTGMenuPushToggle(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	next, err := SetTGMenuPush(Options{CrashDir: crashDir}, "toggle")
	if err != nil {
		t.Fatalf("toggle on: %v", err)
	}
	if next != "ON" {
		t.Fatalf("expected ON, got %s", next)
	}
	next, err = SetTGMenuPush(Options{CrashDir: crashDir}, "toggle")
	if err != nil {
		t.Fatalf("toggle off: %v", err)
	}
	if next != "OFF" {
		t.Fatalf("expected OFF, got %s", next)
	}
}

func TestConfigureTGBotWritesGatewayCfg(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("my_alias=sc\n"), 0o644); err != nil {
		t.Fatalf("write main cfg: %v", err)
	}

	oldHTTP := gatewayHTTPDo
	defer func() { gatewayHTTPDo = oldHTTP }()
	var urls []string
	gatewayHTTPDo = func(req *http.Request) (*http.Response, error) {
		urls = append(urls, req.URL.String())
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	}

	var out bytes.Buffer
	if err := ConfigureTGBot(Options{CrashDir: crashDir}, "tok", "12345678", &out); err != nil {
		t.Fatalf("configure tg: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "gateway.cfg"))
	if err != nil {
		t.Fatalf("read gateway cfg: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "TG_TOKEN=tok") || !strings.Contains(text, "TG_CHATID=12345678") {
		t.Fatalf("unexpected gateway cfg: %s", text)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 telegram calls, got %d", len(urls))
	}
}

func TestRunTGMenuConfigAndToggleStates(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("my_alias=sc\n"), 0o644); err != nil {
		t.Fatalf("write main cfg: %v", err)
	}

	oldHTTP := gatewayHTTPDo
	oldRun := gatewayRunControllerAction
	oldRunning := gatewayIsProcessRunning
	defer func() {
		gatewayHTTPDo = oldHTTP
		gatewayRunControllerAction = oldRun
		gatewayIsProcessRunning = oldRunning
	}()
	gatewayHTTPDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	}
	gatewayRunControllerAction = func(_ string, action string) error { return nil }
	gatewayIsProcessRunning = func(name string) bool { return false }

	input := "2\ntok\n12345678\n1\n3\n0\n"
	var out bytes.Buffer
	if err := RunTGMenu(Options{CrashDir: crashDir}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run tg menu: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "gateway.cfg"))
	if err != nil {
		t.Fatalf("read gateway cfg: %v", err)
	}
	cfg := string(data)
	for _, expect := range []string{
		"TG_TOKEN=tok",
		"TG_CHATID=12345678",
		"bot_tg_service=ON",
		"TG_menupush=ON",
	} {
		if !strings.Contains(cfg, expect) {
			t.Fatalf("expected %q in cfg, got: %s", expect, cfg)
		}
	}
}

func TestRunVmessMenuWritesConfigAndBuildsLink(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	input := "2\n443\n4\n123e4567-e89b-12d3-a456-426614174000\n1\n7\nexample.com\n0\n"
	var out bytes.Buffer
	if err := RunVmessMenu(Options{CrashDir: crashDir}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run vmess menu: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "gateway.cfg"))
	if err != nil {
		t.Fatalf("read gateway cfg: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "vms_port=443") ||
		!strings.Contains(text, "vms_uuid=123e4567-e89b-12d3-a456-426614174000") ||
		!strings.Contains(text, "vms_service=ON") {
		t.Fatalf("unexpected cfg: %s", text)
	}
	if !strings.Contains(out.String(), "vmess://") {
		t.Fatalf("expected vmess link in output, got: %s", out.String())
	}
}

func TestRunShadowsocksMenuWritesConfigAndBuildsLink(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	input := "2\n8388\n3\n1\n1\n5\nexample.com\n0\n"
	var out bytes.Buffer
	if err := RunShadowsocksMenu(Options{CrashDir: crashDir}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run shadowsocks menu: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "gateway.cfg"))
	if err != nil {
		t.Fatalf("read gateway cfg: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "sss_port=8388") ||
		!strings.Contains(text, "sss_cipher=xchacha20-ietf-poly1305") ||
		!strings.Contains(text, "sss_service=ON") {
		t.Fatalf("unexpected cfg: %s", text)
	}
	if !strings.Contains(out.String(), "ss://") {
		t.Fatalf("expected ss link in output, got: %s", out.String())
	}
}

func TestRunTailscaleMenuConfigFlow(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	input := "1\n2\nts-auth\n1\n3\n4\n5\nnode-a\n0\n"
	var out bytes.Buffer
	if err := RunTailscaleMenu(Options{CrashDir: crashDir}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run tailscale menu: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "请先设置秘钥") {
		t.Fatalf("expected missing-key warning, got: %s", text)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "gateway.cfg"))
	if err != nil {
		t.Fatalf("read gateway cfg: %v", err)
	}
	cfg := string(data)
	for _, expect := range []string{
		"ts_auth_key=ts-auth",
		"ts_service=ON",
		"ts_subnet=true",
		"ts_exit_node=true",
		"ts_hostname=node-a",
	} {
		if !strings.Contains(cfg, expect) {
			t.Fatalf("expected %q in cfg, got: %s", expect, cfg)
		}
	}
}

func TestRunWireGuardMenuConfigFlow(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	input := "1\n2\nwg.example.com\n3\n51820\n4\npub-key\n5\npre-key\n6\npri-key\n7\n10.0.0.2/32\n8\nfd7a:115c:a1e0::2/128\n1\n0\n"
	var out bytes.Buffer
	if err := RunWireGuardMenu(Options{CrashDir: crashDir}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("run wireguard menu: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "请先完成必选设置") {
		t.Fatalf("expected required-fields warning, got: %s", text)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "gateway.cfg"))
	if err != nil {
		t.Fatalf("read gateway cfg: %v", err)
	}
	cfg := string(data)
	for _, expect := range []string{
		"wg_server=wg.example.com",
		"wg_port=51820",
		"wg_public_key=pub-key",
		"wg_pre_shared_key=pre-key",
		"wg_private_key=pri-key",
		"wg_ipv4=10.0.0.2/32",
		"wg_ipv6=fd7a:115c:a1e0::2/128",
		"wg_service=ON",
	} {
		if !strings.Contains(cfg, expect) {
			t.Fatalf("expected %q in cfg, got: %s", expect, cfg)
		}
	}
}
