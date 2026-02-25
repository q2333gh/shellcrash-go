package initctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMigratesAndNormalizesConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacyCfg := "clashcore=meta\nclash_v=1\nShellClash=1\nredir_mod=混合模式\nstate=已开启\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellClash.cfg"), []byte(legacyCfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "task.list"), []byte("* * * * * echo 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.srs"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, TmpDir: "/tmp/test-shellcrash"}); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, "configs", "ShellCrash.cfg")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(cfg)
	for _, want := range []string{"crashcore=meta", "core_v=1", "ShellCrash=1", "redir_mod=Mix", "state=ON"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing normalized cfg value %q in %q", want, text)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "yamls", "config.yaml")); err != nil {
		t.Fatalf("expected yamls/config.yaml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "task", "task.list")); err != nil {
		t.Fatalf("expected task/task.list: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "ruleset", "note.srs")); err != nil {
		t.Fatalf("expected ruleset/note.srs: %v", err)
	}
}

func TestRunGeneratesCommandEnvForSingbox(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=singbox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(Options{CrashDir: root, TmpDir: "/tmp/shellcrash-initctl"}); err != nil {
		t.Fatal(err)
	}
	env, err := os.ReadFile(filepath.Join(root, "configs", "command.env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(env)
	for _, want := range []string{
		"TMPDIR=/tmp/shellcrash-initctl",
		"BINDIR=" + root,
		"COMMAND=$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing env %q in %q", want, text)
		}
	}
}

func TestRunInstallsProcdServiceScript(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "proc", "1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "proc", "1", "comm"), []byte("procd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "rc.common"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "starts", "shellcrash.procd"), []byte("#!/bin/sh"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(fsroot, "etc", "init.d", "shellcrash")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected procd service installed: %v", err)
	}
	mode := os.FileMode(0)
	if st, err := os.Stat(dst); err == nil {
		mode = st.Mode().Perm()
	}
	if mode != 0o755 {
		t.Fatalf("expected 0755 mode, got %o", mode)
	}
}

func TestRunInstallsSystemdServiceWithCrashDirReplacement(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "proc", "1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "proc", "1", "comm"), []byte("systemd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	serviceTemplate := "ExecStart=/etc/ShellCrash/start.sh\n"
	if err := os.WriteFile(filepath.Join(root, "starts", "shellcrash.service"), []byte(serviceTemplate), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(fsroot, "etc", "systemd", "system", "shellcrash.service")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("expected systemd service installed: %v", err)
	}
	if strings.Contains(string(data), "/etc/ShellCrash") {
		t.Fatalf("expected crashdir replacement in service file, got: %s", string(data))
	}
	if !strings.Contains(string(data), root) {
		t.Fatalf("expected service file to contain crashdir %q, got: %s", root, string(data))
	}
}

func TestRunFallbackSetsStartOldOnUnknownInit(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("# cfg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}
	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "start_old=ON") {
		t.Fatalf("expected start_old fallback in cfg, got: %s", string(cfgData))
	}
}

func TestRunSystemdProvisionsServiceUser(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "proc", "1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "proc", "1", "comm"), []byte("systemd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "passwd"), []byte("root:x:0:0::/root:/bin/sh\nshellcrash:x:1:1::/home/shellcrash:/bin/false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "group"), []byte("root:x:0:\nshellcrash:x:1:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "starts", "shellcrash.service"), []byte("ExecStart=/etc/ShellCrash/start.sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}
	passwdData, err := os.ReadFile(filepath.Join(fsroot, "etc", "passwd"))
	if err != nil {
		t.Fatal(err)
	}
	passwd := string(passwdData)
	if !strings.Contains(passwd, "shellcrash:x:0:7890::/home/shellcrash:/bin/sh") {
		t.Fatalf("expected systemd shellcrash user line, got: %s", passwd)
	}
	if strings.Contains(passwd, "shellcrash:x:1:1") {
		t.Fatalf("expected stale shellcrash line removed, got: %s", passwd)
	}
	groupData, err := os.ReadFile(filepath.Join(fsroot, "etc", "group"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(groupData), "shellcrash:x:7890:") {
		t.Fatalf("expected shellcrash group, got: %s", string(groupData))
	}
}

func TestRunCleansLegacyShellclashArtifacts(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "etc", "init.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "passwd"), []byte("root:x:0:0::/root:/bin/sh\nshellclash:x:0:7890::/home/shellclash:/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "group"), []byte("root:x:0:\nshellclash:x:7890:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "init.d", "clash"), []byte("# old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}
	passwdData, err := os.ReadFile(filepath.Join(fsroot, "etc", "passwd"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(passwdData), "shellclash:") {
		t.Fatalf("expected shellclash user removed, got: %s", string(passwdData))
	}
	groupData, err := os.ReadFile(filepath.Join(fsroot, "etc", "group"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(groupData), "shellclash:") {
		t.Fatalf("expected shellclash group removed, got: %s", string(groupData))
	}
	if _, err := os.Stat(filepath.Join(fsroot, "etc", "init.d", "clash")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy /etc/init.d/clash removed, err=%v", err)
	}
}

func TestRunPadavanAddsGeneralInitHook(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "etc", "storage"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "starts", "general_init.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initPath := filepath.Join(fsroot, "etc", "storage", "started_script.sh")
	if err := os.WriteFile(initPath, []byte("echo 1\n#ShellCrash初始化旧行\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}

	initData, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(initData)
	wantHook := root + "/starts/general_init.sh & #ShellCrash初始化脚本"
	if strings.Count(text, wantHook) != 1 {
		t.Fatalf("expected one init hook line, got: %q", text)
	}
	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(cfgData)
	if !strings.Contains(cfg, "systype=Padavan") {
		t.Fatalf("expected systype=Padavan, got: %s", cfg)
	}
	if !strings.Contains(cfg, "initdir=/etc/storage/started_script.sh") {
		t.Fatalf("expected initdir persisted, got: %s", cfg)
	}
}

func TestRunContainerAppliesDefaults(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "proc", "1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "proc", "1", "cgroup"), []byte("0::/docker/abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "etc", "profile"), []byte("# profile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}

	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(cfgData)
	for _, want := range []string{
		"systype=container",
		"userguide=1",
		"crashcore=meta",
		"dns_mod=mix",
		"firewall_area=1",
		"firewall_mod=nftables",
		"release_type=master",
		"start_old=OFF",
	} {
		if !strings.Contains(cfg, want) {
			t.Fatalf("missing container default %q in %q", want, cfg)
		}
	}
	profileData, err := os.ReadFile(filepath.Join(fsroot, "etc", "profile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(profileData), root+"/menu.sh") {
		t.Fatalf("expected menu.sh profile hook, got: %s", string(profileData))
	}
	wrapperPath := filepath.Join(fsroot, "usr", "bin", "crash")
	wrapperData, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wrapperData), "CRASHDIR=${CRASHDIR:-"+root+"}") {
		t.Fatalf("unexpected crash wrapper content: %s", string(wrapperData))
	}
}

func TestRunSetsFirewallModeDefaultFromNFTCapability(t *testing.T) {
	oldHas := initctlHasCommand
	initctlHasCommand = func(name string) bool { return name == "nft" }
	defer func() { initctlHasCommand = oldHas }()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: t.TempDir()}); err != nil {
		t.Fatal(err)
	}

	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "firewall_mod=nftables") {
		t.Fatalf("expected firewall_mod=nftables, got: %s", string(cfgData))
	}
}

func TestRunMiSnapshotConfiguresSnapshotInitAndUCI(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "data", "etc", "crontabs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fsroot, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fsroot, "data", "etc", "crontabs", "root"), []byte("# cron"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "starts", "snapshot_init.sh"), []byte("#!/bin/sh\necho snapshot\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	oldHas := initctlHasCommand
	oldRun := initctlRunCommand
	defer func() {
		initctlHasCommand = oldHas
		initctlRunCommand = oldRun
	}()
	initctlHasCommand = func(name string) bool { return name == "uci" }
	calls := make([]string, 0, 8)
	initctlRunCommand = func(name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}

	snapshotPath := filepath.Join(fsroot, "data", "shellcrash_init.sh")
	content, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("expected snapshot init copied to /data: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "CRASHDIR="+root) {
		t.Fatalf("expected pinned CRASHDIR in snapshot init, got: %s", text)
	}
	if _, err := os.Stat(filepath.Join(fsroot, "data", "auto_start.sh")); err != nil {
		t.Fatalf("expected /data/auto_start.sh created: %v", err)
	}

	cfgData, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "systype=mi_snapshot") {
		t.Fatalf("expected systype=mi_snapshot in cfg, got: %s", string(cfgData))
	}

	required := []string{
		"uci set firewall.ShellCrash.path=/data/shellcrash_init.sh",
		"uci set firewall.ShellCrash.enabled=1",
		"uci commit firewall",
	}
	for _, want := range required {
		if !containsString(calls, want) {
			t.Fatalf("missing uci call %q in %v", want, calls)
		}
	}
}

func TestRunNonSnapshotRemovesSnapshotInitWrapper(t *testing.T) {
	root := t.TempDir()
	fsroot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "starts", "snapshot_init.sh")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(Options{CrashDir: root, FSRoot: fsroot}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wrapper); !os.IsNotExist(err) {
		t.Fatalf("expected snapshot init wrapper removed on non-snapshot systems, err=%v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
