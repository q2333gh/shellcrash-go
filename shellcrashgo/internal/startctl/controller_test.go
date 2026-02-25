package startctl

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"shellcrash/internal/coreconfig"
)

func TestCheckControllerAPISuccess(t *testing.T) {
	oldDo := controllerHTTPDo
	defer func() { controllerHTTPDo = oldDo }()
	controllerHTTPDo = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer top-secret" {
			t.Fatalf("missing auth header: %q", req.Header.Get("Authorization"))
		}
		body := io.NopCloser(strings.NewReader(`{"proxies":{}}`))
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}

	c := Controller{
		Cfg: Config{
			DBPort: "9999",
			Secret: "top-secret",
		},
	}
	if !c.checkControllerAPI() {
		t.Fatalf("expected controller API to be ready")
	}
}

func TestRunStartFirewallDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	ctl := Controller{Cfg: Config{CrashDir: crashDir}}
	err := ctl.Run("start_firewall", "", false)
	if err == nil {
		t.Fatal("expected start_firewall error from Go firewall path")
	}
	if strings.Contains(err.Error(), "start_firewall") {
		t.Fatalf("unexpected shell fallback for start_firewall: %v", err)
	}
}

func TestRunStopFirewallDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	ctl := Controller{Cfg: Config{CrashDir: crashDir}}
	err := ctl.Run("stop_firewall", "", false)
	if err == nil {
		t.Fatal("expected stop_firewall error from Go firewall path")
	}
	if strings.Contains(err.Error(), "stop_firewall") {
		t.Fatalf("unexpected shell fallback for stop_firewall: %v", err)
	}
}

func TestRunLegacyFirewallAliasDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	ctl := Controller{Cfg: Config{CrashDir: crashDir}}
	for _, action := range []string{"fw_start", "fw_stop"} {
		err := ctl.Run(action, "", false)
		if err == nil {
			t.Fatalf("expected %s error from Go firewall path", action)
		}
		if strings.Contains(err.Error(), `unsupported action "`) {
			t.Fatalf("unexpected shell fallback for %s: %v", action, err)
		}
	}
}

func TestSplitCommandLineParsesQuotes(t *testing.T) {
	bin, args, err := splitCommandLine(`CrashCore -d "/tmp/A B" -f '/tmp/C D.yaml'`)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "CrashCore" {
		t.Fatalf("unexpected bin: %s", bin)
	}
	got := strings.Join(args, "|")
	want := "-d|/tmp/A B|-f|/tmp/C D.yaml"
	if got != want {
		t.Fatalf("unexpected args: %s", got)
	}
}

func TestSplitCommandLineRejectsBadQuote(t *testing.T) {
	if _, _, err := splitCommandLine(`CrashCore -d "/tmp/A`); err == nil {
		t.Fatal("expected invalid quoting error")
	}
}

func TestStartDetachedCoreWritesPID(t *testing.T) {
	bin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not available")
	}
	td := t.TempDir()
	ctl := Controller{
		Cfg: Config{
			TmpDir:  td,
			Command: bin + " 2",
		},
	}
	if err := ctl.startDetachedCore(bin+" 2", "", true); err != nil {
		t.Fatalf("startDetachedCore failed: %v", err)
	}
	pidData, err := os.ReadFile(filepath.Join(td, "shellcrash.pid"))
	if err != nil {
		t.Fatalf("read shellcrash.pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid pid file content: %q", string(pidData))
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("expected detached process alive, pid=%d err=%v", pid, err)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

func TestCoreExchangeRequiresTarget(t *testing.T) {
	ctl := Controller{}
	err := ctl.RunWithArgs("core_exchange", "", false, nil)
	if err == nil || !strings.Contains(err.Error(), "missing core_exchange target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCoreExchangeUpdatesCoreConfigAndCommandEnv(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	binDir := filepath.Join(crashDir, "bin")
	tmpDir := filepath.Join(crashDir, "tmp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "ShellCrash.cfg")
	envPath := filepath.Join(cfgDir, "command.env")
	if err := os.WriteFile(cfgPath, []byte("crashcore=clash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("COMMAND='old'\nBINDIR='"+binDir+"'\nTMPDIR='"+tmpDir+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(crashDir)
	if err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: cfg}
	if err := ctl.RunWithArgs("core_exchange", "", false, []string{"meta"}); err != nil {
		t.Fatal(err)
	}

	updatedCfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedCfg), "crashcore=meta") {
		t.Fatalf("cfg not updated: %s", string(updatedCfg))
	}
	updatedEnv, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedEnv), "COMMAND='$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml'") {
		t.Fatalf("command.env not updated: %s", string(updatedEnv))
	}
}

func TestRunCheckCoreDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), 0o755); err != nil {
		t.Fatalf("mkdir fake cfg dir: %v", err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			TmpDir:   filepath.Join(td, "tmp"),
			BinDir:   filepath.Join(td, "bin"),
		},
	}
	err := ctl.Run("check_core", "", false)
	if err == nil {
		t.Fatal("expected check_core error from Go core check path")
	}
	if strings.Contains(err.Error(), "check_core") {
		t.Fatalf("unexpected shell fallback for check_core: %v", err)
	}
}

func TestRunStartErrorDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "core_test.log"), []byte("FATAL: test failure\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			TmpDir:   tmpDir,
			BinDir:   binDir,
			StartOld: "ON",
		},
	}
	if err := ctl.Run("start_error", "", false); err != nil {
		t.Fatalf("unexpected start_error failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(crashDir, ".start_error")); err != nil {
		t.Fatalf("expected .start_error marker, err=%v", err)
	}
}

func TestRunCheckNetworkDispatchesToGo(t *testing.T) {
	oldHasCommand := beforeStartHasCommand
	oldPing := beforeStartPing
	defer func() {
		beforeStartHasCommand = oldHasCommand
		beforeStartPing = oldPing
	}()
	beforeStartHasCommand = func(string) bool { return true }
	beforeStartPing = func(string) bool { return true }

	ctl := Controller{Cfg: Config{NetworkCheck: "ON"}}
	if err := ctl.Run("check_network", "", false); err != nil {
		t.Fatalf("unexpected check_network failure: %v", err)
	}
}

func TestRunCheckGeoDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	binDir := filepath.Join(td, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: Config{CrashDir: crashDir, BinDir: binDir}}

	oldDownloader := startctlGeoDownloader
	defer func() { startctlGeoDownloader = oldDownloader }()
	startctlGeoDownloader = func(dst string, remoteName string) error {
		if remoteName != "china_ip_list.txt" {
			t.Fatalf("unexpected remoteName: %s", remoteName)
		}
		return os.WriteFile(dst, []byte(strings.Repeat("x", 64)), 0o644)
	}

	if err := ctl.RunWithArgs("check_geo", "", false, []string{"cn_ip.txt", "china_ip_list.txt"}); err != nil {
		t.Fatalf("unexpected check_geo failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "cn_ip.txt")); err != nil {
		t.Fatalf("expected cn_ip.txt to be created, err=%v", err)
	}
}

func TestRunCheckCNIPDispatchesToGoIPv4(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	binDir := filepath.Join(td, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: Config{CrashDir: crashDir, BinDir: binDir, FirewallMod: "nftables"}}

	oldDownloader := startctlGeoDownloader
	defer func() { startctlGeoDownloader = oldDownloader }()
	startctlGeoDownloader = func(dst string, remoteName string) error {
		if remoteName != "china_ip_list.txt" {
			t.Fatalf("unexpected remoteName: %s", remoteName)
		}
		return os.WriteFile(dst, []byte(strings.Repeat("x", 64)), 0o644)
	}

	if err := ctl.RunWithArgs("check_cnip", "", false, []string{"ipv4"}); err != nil {
		t.Fatalf("unexpected check_cnip ipv4 failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "cn_ip.txt")); err != nil {
		t.Fatalf("expected cn_ip.txt to be created, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "cn_ipv6.txt")); err == nil {
		t.Fatalf("expected cn_ipv6.txt to not be created for ipv4 mode")
	}
}

func TestRunCheckCNIPDispatchesToGoIPv6(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	binDir := filepath.Join(td, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{Cfg: Config{CrashDir: crashDir, BinDir: binDir, FirewallMod: "nftables"}}

	oldDownloader := startctlGeoDownloader
	defer func() { startctlGeoDownloader = oldDownloader }()
	startctlGeoDownloader = func(dst string, remoteName string) error {
		if remoteName != "china_ipv6_list.txt" {
			t.Fatalf("unexpected remoteName: %s", remoteName)
		}
		return os.WriteFile(dst, []byte(strings.Repeat("x", 64)), 0o644)
	}

	if err := ctl.RunWithArgs("check_cnip", "", false, []string{"ipv6"}); err != nil {
		t.Fatalf("unexpected check_cnip ipv6 failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "cn_ipv6.txt")); err != nil {
		t.Fatalf("expected cn_ipv6.txt to be created, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "cn_ip.txt")); err == nil {
		t.Fatalf("expected cn_ip.txt to not be created for ipv6 mode")
	}
}

func TestRunCorePrecheckDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	yamlsDir := filepath.Join(crashDir, "yamls")
	if err := os.MkdirAll(yamlsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(yamlsDir, "config.yaml"), []byte("rules:\n - MATCH,DIRECT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir:  crashDir,
			TmpDir:    filepath.Join(td, "tmp"),
			BinDir:    filepath.Join(td, "bin"),
			CrashCore: "clash",
		},
	}
	err := ctl.Run("core_precheck", "", false)
	if err == nil {
		t.Fatal("expected core_precheck error from Go precheck path")
	}
	if strings.Contains(err.Error(), "core_precheck") {
		t.Fatalf("unexpected shell fallback for core_precheck: %v", err)
	}
}

func TestRunLegacyCorePrecheckAliasDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	yamlsDir := filepath.Join(crashDir, "yamls")
	if err := os.MkdirAll(yamlsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(yamlsDir, "config.yaml"), []byte("rules:\n - MATCH,DIRECT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir:  crashDir,
			TmpDir:    filepath.Join(td, "tmp"),
			BinDir:    filepath.Join(td, "bin"),
			CrashCore: "clash",
		},
	}
	for _, action := range []string{"clash_check", "singbox_check", "clash_config_check", "singbox_config_check", "check_config"} {
		err := ctl.Run(action, "", false)
		if err == nil {
			t.Fatalf("expected %s error from Go precheck path", action)
		}
		if strings.Contains(err.Error(), `unsupported action "`) {
			t.Fatalf("unexpected shell fallback for %s: %v", action, err)
		}
	}
}

func TestRunPrepareRuntimeDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	yamlsDir := filepath.Join(crashDir, "yamls")
	if err := os.MkdirAll(yamlsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(yamlsDir, "config.yaml"), []byte("rules:\n - MATCH,DIRECT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir:  crashDir,
			TmpDir:    filepath.Join(td, "tmp"),
			BinDir:    filepath.Join(td, "bin"),
			CrashCore: "clash",
		},
	}
	err := ctl.Run("prepare_runtime", "", false)
	if err == nil {
		t.Fatal("expected prepare_runtime error from Go precheck path")
	}
	if strings.Contains(err.Error(), "prepare_runtime") {
		t.Fatalf("unexpected shell fallback for prepare_runtime: %v", err)
	}
}

func TestRunHotUpdateDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			TmpDir:   tmpDir,
			DBPort:   "8899",
			Secret:   "top-secret",
		},
	}

	oldCore := controllerCoreConfigRun
	oldHTTP := controllerHTTPDo
	defer func() {
		controllerCoreConfigRun = oldCore
		controllerHTTPDo = oldHTTP
	}()

	coreCalled := false
	controllerCoreConfigRun = func(opts coreconfig.Options) (coreconfig.Result, error) {
		coreCalled = true
		if opts.CrashDir != crashDir {
			t.Fatalf("unexpected crashDir: %s", opts.CrashDir)
		}
		if opts.TmpDir != tmpDir {
			t.Fatalf("unexpected tmpDir: %s", opts.TmpDir)
		}
		return coreconfig.Result{
			Format:     "yaml",
			CoreConfig: filepath.Join(crashDir, "yamls", "config.yaml"),
		}, nil
	}

	httpCalled := false
	controllerHTTPDo = func(req *http.Request) (*http.Response, error) {
		httpCalled = true
		if req.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.String() != "http://127.0.0.1:8899/configs" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		if req.Header.Get("Authorization") != "Bearer top-secret" {
			t.Fatalf("unexpected auth header: %q", req.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if strings.TrimSpace(string(body)) != `{"path":"`+filepath.Join(crashDir, "yamls", "config.yaml")+`"}` {
			t.Fatalf("unexpected body: %s", string(body))
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	if err := ctl.Run("hotupdate", "", false); err != nil {
		t.Fatalf("unexpected hotupdate failure: %v", err)
	}
	if !coreCalled || !httpCalled {
		t.Fatalf("expected core+http called, core=%v http=%v", coreCalled, httpCalled)
	}
}

func TestRunHotUpdateHTTPError(t *testing.T) {
	ctl := Controller{Cfg: Config{DBPort: "8899"}}
	oldCore := controllerCoreConfigRun
	oldHTTP := controllerHTTPDo
	defer func() {
		controllerCoreConfigRun = oldCore
		controllerHTTPDo = oldHTTP
	}()

	controllerCoreConfigRun = func(opts coreconfig.Options) (coreconfig.Result, error) {
		return coreconfig.Result{
			Format:     "yaml",
			CoreConfig: "/tmp/config.yaml",
		}, nil
	}
	controllerHTTPDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("bad config")),
		}, nil
	}

	err := ctl.Run("hotupdate", "", false)
	if err == nil || !strings.Contains(err.Error(), "hotupdate reload failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUnknownActionDispatchesToLegacyScript(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	startsDir := filepath.Join(crashDir, "starts")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(startsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(startsDir, "legacy_custom.sh")
	body := "#!/bin/sh\n" +
		"echo \"$1|$2|$CRASHDIR\" > \"$TMPDIR/legacy_action.out\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			TmpDir:   tmpDir,
			BinDir:   filepath.Join(td, "bin"),
		},
	}
	if err := ctl.RunWithArgs("legacy_custom", "", false, []string{"arg1", "arg2"}); err != nil {
		t.Fatalf("unexpected legacy dispatch failure: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(tmpDir, "legacy_action.out"))
	if err != nil {
		t.Fatalf("read legacy output: %v", err)
	}
	got := strings.TrimSpace(string(out))
	want := "arg1|arg2|" + crashDir
	if got != want {
		t.Fatalf("unexpected legacy script output: got=%q want=%q", got, want)
	}
}

func TestRunUnknownActionWithoutLegacyScriptReturnsError(t *testing.T) {
	ctl := Controller{
		Cfg: Config{
			CrashDir: t.TempDir(),
			TmpDir:   t.TempDir(),
		},
	}
	err := ctl.Run("missing_action", "", false)
	if err == nil || !strings.Contains(err.Error(), `unsupported action "missing_action"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBotTGStartDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	tmpDir := filepath.Join(td, "tmp")
	binDir := filepath.Join(crashDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tgbot := filepath.Join(binDir, "shellcrash-tgbot")
	if err := os.WriteFile(tgbot, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			TmpDir:   tmpDir,
		},
	}
	if err := ctl.Run("bot_tg_start", "", false); err != nil {
		t.Fatalf("unexpected bot_tg_start error: %v", err)
	}
	pidData, err := os.ReadFile(filepath.Join(tmpDir, "bot_tg.pid"))
	if err != nil {
		t.Fatalf("expected bot_tg.pid, err=%v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid bot pid content: %q", string(pidData))
	}
	time.Sleep(100 * time.Millisecond)
	_ = syscall.Kill(pid, syscall.SIGTERM)
}

func TestRunBotTGStopDispatchesToGo(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "bot_tg.pid"), []byte("999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := filepath.Join(t.TempDir(), "crontab.txt")
	if err := os.WriteFile(store, []byte(
		"* * * * * keep\n"+
			"* * * * * old #ShellCrash-TG_BOT守护进程\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
	restoreEnv := installFakeCrontab(t, store)
	defer restoreEnv()

	oldPersistent := cronsetHasPersistentStore
	cronsetHasPersistentStore = func() bool { return false }
	defer func() { cronsetHasPersistentStore = oldPersistent }()

	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			TmpDir:   tmpDir,
		},
	}
	if err := ctl.Run("bot_tg_stop", "", false); err != nil {
		t.Fatalf("unexpected bot_tg_stop error: %v", err)
	}

	out, err := os.ReadFile(store)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if strings.Contains(got, "TG_BOT") {
		t.Fatalf("expected TG_BOT entries removed, got: %s", got)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "bot_tg.pid")); !os.IsNotExist(err) {
		t.Fatalf("expected bot_tg.pid removed, stat err=%v", err)
	}
}

func TestRunBotTGCronDispatchesToGo(t *testing.T) {
	store := filepath.Join(t.TempDir(), "crontab.txt")
	if err := os.WriteFile(store, []byte("0 0 * * * keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restoreEnv := installFakeCrontab(t, store)
	defer restoreEnv()

	oldPersistent := cronsetHasPersistentStore
	cronsetHasPersistentStore = func() bool { return false }
	defer func() { cronsetHasPersistentStore = oldPersistent }()

	crashDir := filepath.Join(t.TempDir(), "ShellCrash")
	ctl := Controller{
		Cfg: Config{
			CrashDir: crashDir,
			BinDir:   "/opt/sc/bin",
		},
	}
	if err := ctl.Run("bot_tg_cron", "", false); err != nil {
		t.Fatalf("unexpected bot_tg_cron error: %v", err)
	}
	out, err := os.ReadFile(store)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "'/opt/sc/bin/shellcrash-startwatchdog' --crashdir ") {
		t.Fatalf("expected startwatchdog cron entry, got: %s", got)
	}
	if !strings.Contains(got, "#ShellCrash-TG_BOT守护进程") {
		t.Fatalf("expected TG_BOT comment marker, got: %s", got)
	}
}
