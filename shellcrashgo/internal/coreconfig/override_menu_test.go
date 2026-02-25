package coreconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunOverrideGroupsMenuAddAndClear(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "yamls"), 0o755); err != nil {
		t.Fatalf("mkdir yamls: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "yamls", "config.yaml"), []byte("proxy-groups:\n  - name: Auto\n"), 0o644); err != nil {
		t.Fatalf("write config yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "yamls", "proxy-groups.yaml"), []byte("#用于添加自定义策略组\n"), 0o644); err != nil {
		t.Fatalf("write proxy groups: %v", err)
	}

	in := strings.NewReader("1\nMyGroup\n1\n1\n3\n1\n0\n")
	var out bytes.Buffer
	if err := RunOverrideGroupsMenu(MenuOptions{CrashDir: root, In: in, Out: &out}); err != nil {
		t.Fatalf("run override groups menu: %v", err)
	}

	groupsPath := filepath.Join(root, "yamls", "proxy-groups.yaml")
	data, err := os.ReadFile(groupsPath)
	if err != nil {
		t.Fatalf("read groups: %v", err)
	}
	if got := string(data); got != "#用于添加自定义策略组\n" {
		t.Fatalf("unexpected groups after clear: %q", got)
	}
}

func TestRunOverrideProxiesMenuAddDeleteAndToggle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "yamls"), 0o755); err != nil {
		t.Fatalf("mkdir yamls: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\nproxies_bypass=OFF\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "yamls", "config.yaml"), []byte("proxy-groups:\n  - name: Auto\n"), 0o644); err != nil {
		t.Fatalf("write config yaml: %v", err)
	}

	input := strings.NewReader("1\nname: \"test\", server: 1.1.1.1, port: 443, type: ss\n1\n2\n1\n4\n1\n0\n")
	var out bytes.Buffer
	if err := RunOverrideProxiesMenu(MenuOptions{CrashDir: root, In: input, Out: &out}); err != nil {
		t.Fatalf("run override proxies menu: %v", err)
	}

	proxiesPath := filepath.Join(root, "yamls", "proxies.yaml")
	data, err := os.ReadFile(proxiesPath)
	if err != nil {
		t.Fatalf("read proxies: %v", err)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected proxies to be empty after delete, got: %q", string(data))
	}

	cfg, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if !strings.Contains(string(cfg), "proxies_bypass=ON\n") {
		t.Fatalf("expected proxies_bypass=ON in cfg, got:\n%s", string(cfg))
	}
}

func TestRunOverrideClashAdvancedCreatesFiles(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if err := RunOverrideClashAdvanced(MenuOptions{CrashDir: root, Out: &out}); err != nil {
		t.Fatalf("run clash advanced: %v", err)
	}

	userPath := filepath.Join(root, "yamls", "user.yaml")
	othersPath := filepath.Join(root, "yamls", "others.yaml")
	if _, err := os.Stat(userPath); err != nil {
		t.Fatalf("user.yaml missing: %v", err)
	}
	if _, err := os.Stat(othersPath); err != nil {
		t.Fatalf("others.yaml missing: %v", err)
	}

	userData, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("read user.yaml: %v", err)
	}
	if !strings.Contains(string(userData), "#port: 7890") {
		t.Fatalf("unexpected user.yaml content:\n%s", string(userData))
	}
}
