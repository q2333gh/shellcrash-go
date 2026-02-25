package snapshotctl

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunTProxyFixCommentsAndRunsSysctl(t *testing.T) {
	root := t.TempDir()
	qca := filepath.Join(root, "etc", "init.d", "qca-nss-ecm")
	if err := os.MkdirAll(filepath.Dir(qca), 0o755); err != nil {
		t.Fatal(err)
	}
	src := "sysctl -w net.bridge.bridge-nf-call-ip\n"
	if err := os.WriteFile(qca, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var ran []string
	err := Run(Options{CrashDir: "/etc/ShellCrash", FSRoot: root, Action: "tproxyfix"}, Deps{
		RunCommand: func(name string, args ...string) error {
			ran = append(ran, strings.Join(append([]string{name}, args...), " "))
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	b, err := os.ReadFile(qca)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "#sysctl -w net.bridge.bridge-nf-call-ip") {
		t.Fatalf("expected script patch, got: %q", string(b))
	}
	if len(ran) != 2 {
		t.Fatalf("expected 2 sysctl calls, got: %v", ran)
	}
}

func TestRunAutoCleanMutatesCronAndRemovesDirs(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"data/etc_bak", "data/usr/log", "data/usr/sec_cfg"} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cron := filepath.Join(root, "etc", "crontabs", "root")
	if err := os.MkdirAll(filepath.Dir(cron), 0o755); err != nil {
		t.Fatal(err)
	}
	cronText := "1 1 * * * /usr/sbin/logrotate\n2 2 * * * /usr/bin/sec_cfg_bak\n"
	if err := os.WriteFile(cron, []byte(cronText), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(Options{CrashDir: "/etc/ShellCrash", FSRoot: root, Action: "auto_clean"}, Deps{
		RunCommand: func(string, ...string) error { return nil },
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	for _, rel := range []string{"data/etc_bak", "data/usr/log", "data/usr/sec_cfg"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected removed %s, err=%v", rel, err)
		}
	}
	b, err := os.ReadFile(cron)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "#ShellCrash自动注释 1 1 * * * /usr/sbin/logrotate") {
		t.Fatalf("expected logrotate line commented, got: %q", got)
	}
	if !strings.Contains(got, "#ShellCrash自动注释 2 2 * * * /usr/bin/sec_cfg_bak") {
		t.Fatalf("expected sec_cfg_bak line commented, got: %q", got)
	}
}

func TestRunTunfixMountAndSymlink(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "data", "ShellCrash")
	tool := filepath.Join(crashDir, "tools", "tun.ko")
	if err := os.MkdirAll(filepath.Dir(tool), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tool, []byte("ko"), 0o644); err != nil {
		t.Fatal(err)
	}
	modPath := "/lib/modules/6.6/ip_tables.ko"
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", "6.6"), 0o755); err != nil {
		t.Fatal(err)
	}

	var mountCmd string
	err := Run(Options{CrashDir: crashDir, FSRoot: root, Action: "tunfix"}, Deps{
		ModInfoIPTables: func() string { return modPath },
		RunCommand: func(name string, args ...string) error {
			mountCmd = strings.Join(append([]string{name}, args...), " ")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !strings.Contains(mountCmd, "mount -o noatime") {
		t.Fatalf("expected mount command, got: %q", mountCmd)
	}
	linkPath := filepath.Join(root, "lib", "modules", "6.6", "tun.ko")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("expected tun symlink, err=%v", err)
	}
	if target != tool {
		t.Fatalf("unexpected symlink target: %q", target)
	}
}

func TestRunInitAppliesAutoSSHParity(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "data", "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "etc", "dropbear"), 0o755); err != nil {
		t.Fatal(err)
	}
	cronPath := filepath.Join(root, "etc", "crontabs", "root")
	if err := os.MkdirAll(filepath.Dir(cronPath), 0o755); err != nil {
		t.Fatal(err)
	}
	cronContent := "* * * * * /etc/ShellCrash/starts/start_legacy_wd.sh shellcrash\n0 5 * * * /usr/bin/echo ok\n"
	if err := os.WriteFile(cronPath, []byte(cronContent), 0o644); err != nil {
		t.Fatal(err)
	}
	dropbearInit := filepath.Join(root, "etc", "init.d", "dropbear")
	if err := os.MkdirAll(filepath.Dir(dropbearInit), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dropbearInit, []byte("channel=\"release\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "mi_mi_autoSSH_pwd=abc123\nredir_mod=redir\n"
	if err := os.WriteFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	keySrc := filepath.Join(crashDir, "configs", "dropbear_rsa_host_key")
	authSrc := filepath.Join(crashDir, "configs", "authorized_keys")
	if err := os.WriteFile(keySrc, []byte("key"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authSrc, []byte("auth"), 0o644); err != nil {
		t.Fatal(err)
	}

	var ran []string
	err := Run(Options{CrashDir: crashDir, FSRoot: root, Action: "init"}, Deps{
		RunCommand: func(name string, args ...string) error {
			ran = append(ran, strings.Join(append([]string{name}, args...), " "))
			return nil
		},
		CommandOutput: func(name string, args ...string) ([]byte, error) {
			cmd := strings.Join(append([]string{name}, args...), " ")
			switch cmd {
			case "uci -c /usr/share/xiaoqiang get xiaoqiang_version.version.CHANNEL":
				return []byte("beta\n"), nil
			case "netstat -ntul":
				return []byte("tcp 0 0 0.0.0.0:80 0.0.0.0:* LISTEN\n"), nil
			case "nvram get ssh_en":
				return []byte("0\n"), nil
			case "nvram get telnet_en":
				return []byte("0\n"), nil
			default:
				return nil, errors.New("unexpected command")
			}
		},
		IsProcessRunning: func(name string) bool { return false },
		HasLANInterface:  func(string) bool { return true },
		Sleep:            func(time.Duration) {},
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !containsCommand(ran, "uci -c /usr/share/xiaoqiang set xiaoqiang_version.version.CHANNEL=stable") {
		t.Fatalf("expected uci set stable, got: %v", ran)
	}
	if !containsCommand(ran, filepath.Join(root, "etc", "init.d", "shellcrash")+" disable") {
		t.Fatalf("expected shellcrash disable, got: %v", ran)
	}
	if !containsCommand(ran, filepath.Join(root, "etc", "init.d", "dropbear")+" restart") {
		t.Fatalf("expected dropbear restart, got: %v", ran)
	}
	if !containsCommand(ran, "sh -c printf '%s\\n%s\\n' 'abc123' 'abc123' | passwd root") {
		t.Fatalf("expected passwd restore command, got: %v", ran)
	}
	if !containsCommand(ran, "nvram set ssh_en=1") || !containsCommand(ran, "nvram set telnet_en=1") {
		t.Fatalf("expected nvram set commands, got: %v", ran)
	}
	if !containsCommand(ran, "nvram commit") {
		t.Fatalf("expected nvram commit, got: %v", ran)
	}

	dropbearData, err := os.ReadFile(dropbearInit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dropbearData), `channel="debug"`) {
		t.Fatalf("dropbear channel should be debug, got: %q", string(dropbearData))
	}
	keyDst := filepath.Join(root, "etc", "dropbear", "dropbear_rsa_host_key")
	keyLink, err := os.Readlink(keyDst)
	if err != nil {
		t.Fatalf("expected key symlink: %v", err)
	}
	if keyLink != keySrc {
		t.Fatalf("unexpected key symlink target: %q", keyLink)
	}

	cronData, err := os.ReadFile(cronPath)
	if err != nil {
		t.Fatal(err)
	}
	cronAfter := string(cronData)
	if strings.Contains(cronAfter, "start_legacy_wd.sh shellcrash") {
		t.Fatalf("legacy watchdog cron line should be pruned, got: %q", cronAfter)
	}
	if !strings.Contains(cronAfter, "/usr/bin/echo ok") {
		t.Fatalf("non-legacy cron lines should remain, got: %q", cronAfter)
	}
}

func TestRunInitWaitsForConfigAndErrorsOnTimeout(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "data", "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sleepCalls := 0
	err := Run(Options{CrashDir: crashDir, FSRoot: root, Action: "init"}, Deps{
		RunCommand:       func(string, ...string) error { return nil },
		CommandOutput:    func(string, ...string) ([]byte, error) { return nil, errors.New("missing") },
		HasLANInterface:  func(string) bool { return true },
		IsProcessRunning: func(string) bool { return false },
		Sleep: func(time.Duration) {
			sleepCalls++
		},
	})
	if err == nil || !strings.Contains(err.Error(), "timeout waiting for config") {
		t.Fatalf("expected timeout waiting for config error, got: %v", err)
	}
	if sleepCalls != 21 {
		t.Fatalf("expected 21 config wait sleeps, got: %d", sleepCalls)
	}
}

func containsCommand(all []string, want string) bool {
	for _, cmd := range all {
		if cmd == want {
			return true
		}
	}
	return false
}
