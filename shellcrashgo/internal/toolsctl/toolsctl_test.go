package toolsctl

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"shellcrash/internal/coreconfig"
	"shellcrash/internal/snapshotctl"
)

func TestRunSSHToolsMenuSetPortWritesConfigAndCleansFirewallMarker(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("ssh_port=10022\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	firewallUser := filepath.Join(td, "firewall.user")
	if err := os.WriteFile(firewallUser, []byte("iptables x #启用外网访问SSH服务\n"), 0o644); err != nil {
		t.Fatalf("write firewall.user: %v", err)
	}

	origExec := toolsExecCommand
	defer func() { toolsExecCommand = origExec }()
	toolsExecCommand = func(name string, args ...string) error { return nil }
	origPortInUse := toolsPortInUse
	defer func() { toolsPortInUse = origPortInUse }()
	toolsPortInUse = func(port int) bool { return false }

	in := strings.NewReader("1\n10023\n0\n")
	var out strings.Builder
	if err := RunSSHToolsMenu(Options{CrashDir: crashDir, FirewallUserPath: firewallUser}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if !strings.Contains(string(cfgOut), "ssh_port=10023") {
		t.Fatalf("expected updated ssh_port, got=%s", string(cfgOut))
	}

	fwOut, err := os.ReadFile(firewallUser)
	if err != nil {
		t.Fatalf("read firewall.user: %v", err)
	}
	if strings.Contains(string(fwOut), "启用外网访问SSH服务") {
		t.Fatalf("expected firewall marker removed, got=%s", string(fwOut))
	}
}

func TestRunSSHToolsMenuToggleOnAppendsFirewallRules(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("ssh_port=10022\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	firewallUser := filepath.Join(td, "firewall.user")
	if err := os.WriteFile(firewallUser, []byte(""), 0o644); err != nil {
		t.Fatalf("write firewall.user: %v", err)
	}

	origExec := toolsExecCommand
	defer func() { toolsExecCommand = origExec }()
	toolsExecCommand = func(name string, args ...string) error { return nil }

	in := strings.NewReader("3\n")
	var out strings.Builder
	if err := RunSSHToolsMenu(Options{CrashDir: crashDir, FirewallUserPath: firewallUser}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	fwOut, err := os.ReadFile(firewallUser)
	if err != nil {
		t.Fatalf("read firewall.user: %v", err)
	}
	if !strings.Contains(string(fwOut), "#启用外网访问SSH服务") {
		t.Fatalf("expected firewall marker written, got=%s", string(fwOut))
	}
}

func TestRunMiAutoSSHPersistsConfigAndCopiesDropbearFiles(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("foo=bar\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	dropbearDir := filepath.Join(td, "dropbear")
	if err := os.MkdirAll(dropbearDir, 0o755); err != nil {
		t.Fatalf("mkdir dropbear: %v", err)
	}
	keyPath := filepath.Join(dropbearDir, "dropbear_rsa_host_key")
	authPath := filepath.Join(dropbearDir, "authorized_keys")
	if err := os.WriteFile(keyPath, []byte("key-data"), 0o644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.WriteFile(authPath, []byte("auth-data"), 0o644); err != nil {
		t.Fatalf("write auth: %v", err)
	}

	origExec := toolsExecCommand
	defer func() { toolsExecCommand = origExec }()
	calls := 0
	toolsExecCommand = func(name string, args ...string) error {
		calls++
		return nil
	}

	in := strings.NewReader("mypassword\n")
	var out strings.Builder
	err := RunMiAutoSSH(Options{
		CrashDir:        crashDir,
		DropbearKeyPath: keyPath,
		AuthKeysPath:    authPath,
	}, in, &out)
	if err != nil {
		t.Fatalf("run mi-auto-ssh: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	gotCfg := string(cfgOut)
	if !strings.Contains(gotCfg, "mi_mi_autoSSH=已配置") {
		t.Fatalf("expected mi_mi_autoSSH persisted, got=%s", gotCfg)
	}
	if !strings.Contains(gotCfg, "mi_mi_autoSSH_pwd=mypassword") {
		t.Fatalf("expected mi_mi_autoSSH_pwd persisted, got=%s", gotCfg)
	}
	if !strings.Contains(gotCfg, "foo=bar") {
		t.Fatalf("expected existing cfg entries preserved, got=%s", gotCfg)
	}

	copiedKey, err := os.ReadFile(filepath.Join(cfgDir, "dropbear_rsa_host_key"))
	if err != nil {
		t.Fatalf("read copied key: %v", err)
	}
	if string(copiedKey) != "key-data" {
		t.Fatalf("unexpected copied key content: %q", string(copiedKey))
	}
	copiedAuth, err := os.ReadFile(filepath.Join(cfgDir, "authorized_keys"))
	if err != nil {
		t.Fatalf("read copied auth: %v", err)
	}
	if string(copiedAuth) != "auth-data" {
		t.Fatalf("unexpected copied auth content: %q", string(copiedAuth))
	}

	if calls != 0 {
		t.Fatalf("expected no nvram calls on non-nvram hosts, got=%d", calls)
	}
}

func TestToggleMiOtaUpdateDisablesActiveCrontab(t *testing.T) {
	td := t.TempDir()
	cron := filepath.Join(td, "root")
	if err := os.WriteFile(cron, []byte("15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1\n"), 0o644); err != nil {
		t.Fatalf("write crontab: %v", err)
	}

	action, err := ToggleMiOtaUpdate(Options{CrontabPath: cron})
	if err != nil {
		t.Fatalf("toggle ota: %v", err)
	}
	if action != "禁用" {
		t.Fatalf("unexpected action: %q", action)
	}

	got, err := os.ReadFile(cron)
	if err != nil {
		t.Fatalf("read crontab: %v", err)
	}
	if !strings.Contains(string(got), "#15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1") {
		t.Fatalf("expected commented ota line, got=%s", string(got))
	}
}

func TestToggleMiOtaUpdateEnablesWhenOnlyCommentedLineExists(t *testing.T) {
	td := t.TempDir()
	cron := filepath.Join(td, "root")
	if err := os.WriteFile(cron, []byte("#15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1\n"), 0o644); err != nil {
		t.Fatalf("write crontab: %v", err)
	}

	action, err := ToggleMiOtaUpdate(Options{CrontabPath: cron})
	if err != nil {
		t.Fatalf("toggle ota: %v", err)
	}
	if action != "启用" {
		t.Fatalf("unexpected action: %q", action)
	}

	got, err := os.ReadFile(cron)
	if err != nil {
		t.Fatalf("read crontab: %v", err)
	}
	if !strings.Contains(string(got), "15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1") {
		t.Fatalf("expected enabled ota line, got=%s", string(got))
	}
	if strings.Contains(string(got), "#15 3,4,5 * * * /usr/sbin/otapredownload >/dev/null 2>&1") {
		t.Fatalf("expected commented ota line removed, got=%s", string(got))
	}
}

func TestRunLogPusherMenuSetsTelegramPush(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("foo=bar\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("1\ntg-token\n123456\n0\n")
	var out strings.Builder
	if err := RunLogPusherMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run log-pusher: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	text := string(cfgOut)
	if !strings.Contains(text, "push_TG=tg-token") {
		t.Fatalf("expected telegram token persisted, got=%s", text)
	}
	if !strings.Contains(text, "chat_ID=123456") {
		t.Fatalf("expected chat_ID persisted, got=%s", text)
	}
}

func TestRunLogPusherMenuToggleTaskPush(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("task_push=1\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("b\n0\n")
	var out strings.Builder
	if err := RunLogPusherMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run log-pusher: %v", err)
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if !strings.Contains(string(cfgOut), "task_push=") {
		t.Fatalf("expected task_push toggled off, got=%s", string(cfgOut))
	}
}

func TestRunLogPusherMenuClearLogFile(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("foo=bar\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	logPath := filepath.Join(tmpDir, "ShellCrash.log")
	if err := os.WriteFile(logPath, []byte("line1\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	in := strings.NewReader("d\n0\n")
	var out strings.Builder
	if err := RunLogPusherMenu(Options{CrashDir: crashDir, TmpDir: tmpDir}, in, &out); err != nil {
		t.Fatalf("run log-pusher: %v", err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected log removed, err=%v", err)
	}
}

func TestToggleMiTunfixDisablesWhenPatchExists(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	toolDir := filepath.Join(crashDir, "tools")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("mkdir tools: %v", err)
	}
	patch := filepath.Join(toolDir, "tun.ko")
	if err := os.WriteFile(patch, []byte("x"), 0o644); err != nil {
		t.Fatalf("write patch: %v", err)
	}

	action, err := ToggleMiTunfix(Options{CrashDir: crashDir})
	if err != nil {
		t.Fatalf("toggle tunfix: %v", err)
	}
	if action != "disabled" {
		t.Fatalf("unexpected action: %q", action)
	}
	if _, err := os.Stat(patch); !os.IsNotExist(err) {
		t.Fatalf("expected patch removed, err=%v", err)
	}
}

func TestRunTestCommandMenuShowCoreConfigTop40(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	jsonDir := filepath.Join(crashDir, "jsons")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		t.Fatalf("mkdir json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=singbox\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	var lines []string
	for i := 1; i <= 45; i++ {
		lines = append(lines, "line-"+strconv.Itoa(i))
	}
	if err := os.WriteFile(filepath.Join(jsonDir, "config.json"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write json config: %v", err)
	}

	in := strings.NewReader("5\n0\n")
	var out strings.Builder
	if err := RunTestCommandMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run testcommand: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "line-1") || !strings.Contains(text, "line-40") {
		t.Fatalf("expected first 40 lines in output, got=%s", text)
	}
	if strings.Contains(text, "line-41") {
		t.Fatalf("unexpected line beyond top 40 in output, got=%s", text)
	}
}

func TestRunTestCommandMenuDebugDispatchesToGoDebugMenu(t *testing.T) {
	orig := toolsRunDebugMenu
	defer func() { toolsRunDebugMenu = orig }()
	called := false
	toolsRunDebugMenu = func(opts Options, in io.Reader, out io.Writer) error {
		called = true
		return nil
	}

	in := strings.NewReader("1\n0\n")
	var out strings.Builder
	if err := RunTestCommandMenu(Options{CrashDir: t.TempDir()}, in, &out); err != nil {
		t.Fatalf("run testcommand: %v", err)
	}
	if !called {
		t.Fatal("expected option 1 to dispatch to Go debug menu")
	}
}

func TestRunDebugMenuLevelErrorUsesStartctl(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	orig := toolsStartctlRun
	defer func() { toolsStartctlRun = orig }()
	var got []string
	toolsStartctlRun = func(dir string, args ...string) error {
		got = append([]string{dir}, args...)
		return nil
	}

	in := strings.NewReader("3\n")
	var out strings.Builder
	if err := RunDebugMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run debug menu: %v", err)
	}
	if strings.Join(got, " ") != crashDir+" debug error" {
		t.Fatalf("unexpected startctl call: %v", got)
	}
}

func TestRunDebugMenuOption8UsesGoStartctlDebug(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origStartctl := toolsStartctlRun
	origExecCmd := toolsExecCmd
	defer func() {
		toolsStartctlRun = origStartctl
		toolsExecCmd = origExecCmd
	}()

	var got []string
	toolsStartctlRun = func(dir string, args ...string) error {
		got = append([]string{dir}, args...)
		return nil
	}
	execCalled := false
	toolsExecCmd = func(stdin io.Reader, out io.Writer, name string, args ...string) error {
		execCalled = true
		return nil
	}

	in := strings.NewReader("8\n")
	var out strings.Builder
	if err := RunDebugMenu(Options{CrashDir: crashDir}, in, &out); err != nil {
		t.Fatalf("run debug menu: %v", err)
	}
	if strings.Join(got, " ") != crashDir+" debug" {
		t.Fatalf("unexpected startctl call: %v", got)
	}
	if execCalled {
		t.Fatal("expected option 8 not to shell out via toolsExecCmd")
	}
}

func TestToggleMiTunfixEnablesAndRunsSnapshotTunfix(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	srcDir := filepath.Join(crashDir, "bin", "fix")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	srcPatch := filepath.Join(srcDir, "tun.ko")
	if err := os.WriteFile(srcPatch, []byte("tun-data"), 0o644); err != nil {
		t.Fatalf("write src patch: %v", err)
	}

	binDir := filepath.Join(td, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	modinfo := filepath.Join(binDir, "modinfo")
	if err := os.WriteFile(modinfo, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake modinfo: %v", err)
	}

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+":"+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath)

	origSnapshotRun := toolsSnapshotRun
	defer func() { toolsSnapshotRun = origSnapshotRun }()
	called := false
	toolsSnapshotRun = func(opts snapshotctl.Options, deps snapshotctl.Deps) error {
		called = true
		if opts.CrashDir != crashDir {
			t.Fatalf("unexpected crashdir: %s", opts.CrashDir)
		}
		if opts.Action != "tunfix" {
			t.Fatalf("unexpected action: %s", opts.Action)
		}
		return nil
	}

	action, err := ToggleMiTunfix(Options{CrashDir: crashDir})
	if err != nil {
		t.Fatalf("toggle tunfix: %v", err)
	}
	if action != "enabled" {
		t.Fatalf("unexpected action: %q", action)
	}
	if !called {
		t.Fatal("expected snapshot tunfix run")
	}

	targetPatch := filepath.Join(crashDir, "tools", "tun.ko")
	got, err := os.ReadFile(targetPatch)
	if err != nil {
		t.Fatalf("read target patch: %v", err)
	}
	if string(got) != "tun-data" {
		t.Fatalf("unexpected target patch content: %q", string(got))
	}
}

func TestRunToolsMenuDispatchesTestCommand(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("systype=normal\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origRunTest := toolsRunTestCommandMenu
	origHasOta := toolsHasMiOtaBinary
	defer func() {
		toolsRunTestCommandMenu = origRunTest
		toolsHasMiOtaBinary = origHasOta
	}()
	called := false
	toolsRunTestCommandMenu = func(opts Options, in io.Reader, out io.Writer) error {
		called = true
		return nil
	}
	toolsHasMiOtaBinary = func() bool { return false }

	in := strings.NewReader("1\n0\n")
	var out strings.Builder
	if err := RunToolsMenu(Options{
		CrashDir:         crashDir,
		FirewallUserPath: filepath.Join(td, "firewall.user"),
		CrontabPath:      filepath.Join(td, "root.cron"),
	}, in, &out); err != nil {
		t.Fatalf("run tools menu: %v", err)
	}
	if !called {
		t.Fatal("expected testcommand menu to be dispatched")
	}
}

func TestRunToolsMenuMiAutoSSHUnsupportedOnNonMiSnapshot(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("systype=normal\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origRunMiAuto := toolsRunMiAutoSSHMenu
	origHasOta := toolsHasMiOtaBinary
	defer func() {
		toolsRunMiAutoSSHMenu = origRunMiAuto
		toolsHasMiOtaBinary = origHasOta
	}()
	called := false
	toolsRunMiAutoSSHMenu = func(opts Options, in io.Reader, out io.Writer) error {
		called = true
		return nil
	}
	toolsHasMiOtaBinary = func() bool { return false }

	in := strings.NewReader("6\n0\n")
	var out strings.Builder
	if err := RunToolsMenu(Options{
		CrashDir:         crashDir,
		FirewallUserPath: filepath.Join(td, "firewall.user"),
		CrontabPath:      filepath.Join(td, "root.cron"),
	}, in, &out); err != nil {
		t.Fatalf("run tools menu: %v", err)
	}
	if called {
		t.Fatal("expected mi-auto-ssh not to run on unsupported systype")
	}
	if !strings.Contains(out.String(), "不支持的设备！") {
		t.Fatalf("expected unsupported message, got=%s", out.String())
	}
}

func TestRunToolsMenuDispatchesUserguide(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("systype=normal\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origRunGuide := toolsRunUserguide
	origHasOta := toolsHasMiOtaBinary
	defer func() {
		toolsRunUserguide = origRunGuide
		toolsHasMiOtaBinary = origHasOta
	}()
	called := false
	toolsRunUserguide = func(opts Options, in io.Reader, out io.Writer) error {
		called = true
		return nil
	}
	toolsHasMiOtaBinary = func() bool { return false }

	in := strings.NewReader("2\n")
	var out strings.Builder
	if err := RunToolsMenu(Options{
		CrashDir:         crashDir,
		FirewallUserPath: filepath.Join(td, "firewall.user"),
		CrontabPath:      filepath.Join(td, "root.cron"),
	}, in, &out); err != nil {
		t.Fatalf("run tools menu: %v", err)
	}
	if !called {
		t.Fatal("expected option 2 to dispatch to Go userguide")
	}
}

func TestRunUserguideLocalProxyProfileAndImportConfig(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("cputype=linux-amd64\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origApply := toolsTaskApplyRecommended
	origCore := toolsCoreConfigRun
	defer func() {
		toolsTaskApplyRecommended = origApply
		toolsCoreConfigRun = origCore
	}()

	recommendedCalled := false
	toolsTaskApplyRecommended = func(dir string) error {
		recommendedCalled = true
		if dir != crashDir {
			t.Fatalf("unexpected crashdir: %s", dir)
		}
		return nil
	}
	coreCalled := false
	toolsCoreConfigRun = func(opts coreconfig.Options) (coreconfig.Result, error) {
		coreCalled = true
		if opts.CrashDir != crashDir {
			t.Fatalf("unexpected crashdir: %s", opts.CrashDir)
		}
		return coreconfig.Result{}, nil
	}

	in := strings.NewReader("2\n1\n")
	var out strings.Builder
	if err := RunUserguide(Options{CrashDir: crashDir, TmpDir: filepath.Join(td, "tmp")}, in, &out); err != nil {
		t.Fatalf("run userguide: %v", err)
	}

	if !recommendedCalled {
		t.Fatal("expected recommended tasks to be applied")
	}
	if !coreCalled {
		t.Fatal("expected config import flow to invoke coreconfig")
	}

	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	cfgText := string(cfgOut)
	if !strings.Contains(cfgText, "redir_mod=Redir") {
		t.Fatalf("expected redir_mod set for local proxy profile, got=%s", cfgText)
	}
	if !strings.Contains(cfgText, "common_ports=OFF") {
		t.Fatalf("expected common_ports OFF, got=%s", cfgText)
	}
	if !strings.Contains(cfgText, "firewall_area=2") {
		t.Fatalf("expected firewall_area=2, got=%s", cfgText)
	}
}

func TestSplitCommandLineParsesQuotedArgs(t *testing.T) {
	cmd, args, err := splitCommandLine(`"/tmp/Crash Core" -f "/tmp/config file.yaml"`)
	if err != nil {
		t.Fatalf("split command failed: %v", err)
	}
	if cmd != "/tmp/Crash Core" {
		t.Fatalf("unexpected command: %q", cmd)
	}
	if len(args) != 2 || args[0] != "-f" || args[1] != "/tmp/config file.yaml" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestSplitCommandLineRejectsInvalidQuote(t *testing.T) {
	_, _, err := splitCommandLine(`"/tmp/CrashCore -f /tmp/config.yaml`)
	if err == nil {
		t.Fatal("expected invalid quoting error")
	}
}

func TestFilterNftRouteOutputRemovesCNIPSetsAndClosingBraces(t *testing.T) {
	input := strings.Join([]string{
		"table inet shellcrash {",
		"    set cn_ip {",
		"        type ipv4_addr",
		"        elements = { 1.1.1.0/24 }",
		"    }",
		"    set cn_ip6 {",
		"        type ipv6_addr",
		"    }",
		"    chain prerouting {",
		"        type nat hook prerouting priority -100;",
		"    }",
		"}",
	}, "\n")
	got := filterNftRouteOutput(input)
	if strings.Contains(got, "set cn_ip {") || strings.Contains(got, "set cn_ip6 {") {
		t.Fatalf("expected cn_ip sets removed, got:\n%s", got)
	}
	if strings.Contains(got, "\n}") || strings.Contains(got, "\n    }") {
		t.Fatalf("expected standalone closing braces removed, got:\n%s", got)
	}
	if !strings.Contains(got, "chain prerouting {") {
		t.Fatalf("expected other table content kept, got:\n%s", got)
	}
}

func TestRunShowRouteRulesNFTUsesGoFilter(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("firewall_mod=nftables\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	origOut := toolsCommandOutput
	origExec := toolsExecCmd
	defer func() {
		toolsCommandOutput = origOut
		toolsExecCmd = origExec
	}()
	toolsCommandOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("table inet shellcrash {\n    set cn_ip {\n        type ipv4_addr\n    }\n    chain output {\n    }\n}\n"), nil
	}
	execCalled := false
	toolsExecCmd = func(stdin io.Reader, out io.Writer, name string, args ...string) error {
		execCalled = true
		return nil
	}

	var out strings.Builder
	runShowRouteRules(Options{CrashDir: crashDir}, &out)
	text := out.String()
	if strings.Contains(text, "set cn_ip") {
		t.Fatalf("expected set cn_ip removed, got:\n%s", text)
	}
	if !strings.Contains(text, "chain output {") {
		t.Fatalf("expected chain output kept, got:\n%s", text)
	}
	if execCalled {
		t.Fatal("expected nftables path not to shell out")
	}
}
