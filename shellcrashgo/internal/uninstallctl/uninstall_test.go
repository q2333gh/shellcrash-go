package uninstallctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunKeepConfigRemovesRuntimeContentAndCleansProfiles(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "etc", "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"configs", "yamls", "jsons", "bin", "task"} {
		if err := os.MkdirAll(filepath.Join(crashDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), []byte("x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "bin", "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	etcProfile := filepath.Join(root, "etc", "profile")
	if err := os.MkdirAll(filepath.Dir(etcProfile), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := strings.Join([]string{
		"alias sc='/x/menu.sh'",
		"alias crash='/x/menu.sh'",
		"export CRASHDIR=/x",
		"all_proxy=http://127.0.0.1:7890",
		"PATH=/usr/bin",
		"",
	}, "\n")
	if err := os.WriteFile(etcProfile, []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	firewallUser := filepath.Join(root, "etc", "firewall.user")
	if err := os.WriteFile(firewallUser, []byte("# 启用外网访问SSH服务\nkeep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	passwdPath := filepath.Join(root, "etc", "passwd")
	if err := os.WriteFile(passwdPath, []byte("root:x:0:0::/root:/bin/sh\nshellcrash:x:0:7890::/home/shellcrash:/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	groupPath := filepath.Join(root, "etc", "group")
	if err := os.WriteFile(groupPath, []byte("root:x:0:\nshellcrash:x:7890:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(root, "opt", "scbin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var called []string
	err := Run(Options{
		CrashDir:   crashDir,
		BinDir:     binDir,
		Alias:      "sc",
		KeepConfig: true,
		FSRoot:     root,
	}, Deps{
		StartAction: func(crashDir, action string, args []string) error {
			called = append(called, action+" "+strings.Join(args, " "))
			return nil
		},
		RunCommand: func(string, ...string) error { return nil },
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(crashDir, "configs")); err != nil {
		t.Fatalf("expected configs kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(crashDir, "bin")); !os.IsNotExist(err) {
		t.Fatalf("expected bin removed, err=%v", err)
	}
	if _, err := os.Stat(binDir); !os.IsNotExist(err) {
		t.Fatalf("expected bindir removed, err=%v", err)
	}
	gotProfile, err := os.ReadFile(etcProfile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(gotProfile), "alias sc=") || strings.Contains(string(gotProfile), "all_proxy") {
		t.Fatalf("expected aliases/proxy removed, got: %s", string(gotProfile))
	}
	gotPasswd, err := os.ReadFile(passwdPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(gotPasswd), "shellcrash:") {
		t.Fatalf("expected shellcrash user removed, got: %s", string(gotPasswd))
	}
	if !containsPrefix(called, "stop ") || !containsPrefix(called, "cronset clash服务") {
		t.Fatalf("expected stop+cronset actions, got: %v", called)
	}
}

func TestRunRemoveAllDeletesCrashDir(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "data", "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Run(Options{CrashDir: crashDir, FSRoot: root}, Deps{
		StartAction: func(string, string, []string) error { return nil },
		RunCommand:  func(string, ...string) error { return nil },
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if _, err := os.Stat(crashDir); !os.IsNotExist(err) {
		t.Fatalf("expected crashdir removed, err=%v", err)
	}
}

func TestRunRejectsSlashCrashDir(t *testing.T) {
	err := Run(Options{CrashDir: "/"}, Deps{})
	if err == nil || !strings.Contains(err.Error(), "invalid crashdir") {
		t.Fatalf("expected invalid crashdir error, got: %v", err)
	}
}

func containsPrefix(items []string, prefix string) bool {
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}
