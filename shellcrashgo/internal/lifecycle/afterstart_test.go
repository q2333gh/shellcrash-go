package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAfterStartCoreFlow(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(filepath.Join(crashDir, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(crashDir, "task", "cron"), []byte("0 1 * * * echo a\n"), 0o644)
	_ = os.WriteFile(filepath.Join(crashDir, "task", "running"), []byte("0 1 * * * echo a\n0 2 * * * echo b\n"), 0o644)
	_ = os.WriteFile(filepath.Join(crashDir, "task", "afstart"), []byte("echo hi\n"), 0o644)

	var slept time.Duration
	startedFW := false
	taskRan := ""
	var wrote []string
	err := AfterStart(AfterStartOptions{
		CrashDir:      crashDir,
		TmpDir:        tmpDir,
		StartOld:      "ON",
		StartDelaySec: 3,
		BotTGService:  "ON",
	}, AfterStartDeps{
		Sleep:         func(d time.Duration) { slept = d },
		Now:           func() time.Time { return time.Unix(12345, 0) },
		IsCoreRunning: func() bool { return true },
		StartFirewall: func() error { startedFW = true; return nil },
		ReadSystemCron: func() ([]string, error) {
			return []string{"@reboot echo c", "0 1 * * * echo a"}, nil
		},
		WriteSystemCron: func(lines []string) error { wrote = append([]string{}, lines...); return nil },
		RunTaskScript:   func(path string) error { taskRan = path; return nil },
		InjectFirewallFn: func(string, string) error {
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if slept != 3*time.Second {
		t.Fatalf("expected delay, got %v", slept)
	}
	if !startedFW {
		t.Fatalf("firewall should be started")
	}
	marker, err := os.ReadFile(filepath.Join(tmpDir, "crash_start_time"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(marker)) != "12345" {
		t.Fatalf("unexpected marker content: %q", string(marker))
	}
	if taskRan != filepath.Join(crashDir, "task", "afstart") {
		t.Fatalf("afstart task not run: %q", taskRan)
	}
	joined := strings.Join(wrote, "\n")
	if !strings.Contains(joined, "shellcrash-startwatchdog") || !strings.Contains(joined, "#ShellCrash保守模式守护进程") {
		t.Fatalf("missing start_old watchdog cron line: %v", wrote)
	}
	if !strings.Contains(joined, "'bot_tg' #ShellCrash-TG_BOT守护进程") {
		t.Fatalf("missing bot_tg watchdog cron line: %v", wrote)
	}
	if strings.Contains(joined, "start_legacy_wd.sh") {
		t.Fatalf("legacy watchdog script should not be referenced: %v", wrote)
	}
	if strings.Count(joined, "0 1 * * * echo a") != 1 {
		t.Fatalf("duplicate cron entries were not deduped: %v", wrote)
	}
}

func TestAfterStartCoreNotRunning(t *testing.T) {
	err := AfterStart(AfterStartOptions{CrashDir: "/x", TmpDir: "/y"}, AfterStartDeps{
		IsCoreRunning: func() bool { return false },
		StartFirewall: func() error { return nil },
	})
	if err != ErrCoreNotRunning {
		t.Fatalf("expected ErrCoreNotRunning, got %v", err)
	}
}

func TestInjectFirewallTaskHook(t *testing.T) {
	td := t.TempDir()
	fw := filepath.Join(td, "firewall")
	content := "#!/bin/sh\nfw3 restart\necho x\nfw4 start\n"
	if err := os.WriteFile(fw, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	task := filepath.Join(td, "affirewall")
	if err := InjectFirewallTaskHook(fw, task); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(fw)
	txt := string(b)
	if strings.Count(txt, ". "+task) != 2 {
		t.Fatalf("expected inserted hook twice, got %q", txt)
	}
	if err := InjectFirewallTaskHook(fw, task); err != nil {
		t.Fatal(err)
	}
	b2, _ := os.ReadFile(fw)
	if string(b2) != txt {
		t.Fatalf("injection should be idempotent")
	}
}

func TestGeneralInit(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("my_alias=sc\nshtype=bash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := GeneralInit(td, GeneralInitDeps{Start: func() error { called = true; return nil }}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatalf("start should be called")
	}
	called = false
	_ = os.WriteFile(filepath.Join(td, ".dis_startup"), []byte("1"), 0o644)
	var cronWritten []string
	if err := GeneralInit(td, GeneralInitDeps{Start: func() error { called = true; return nil }}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatalf("start should not be called when disabled")
	}
	if err := GeneralInit(td, GeneralInitDeps{
		Start: func() error { called = true; return nil },
		ReadSystemCron: func() ([]string, error) {
			return []string{"* * * * * run #ShellCrash保守模式守护进程", "0 1 * * * echo ok"}, nil
		},
		WriteSystemCron: func(lines []string) error {
			cronWritten = append([]string{}, lines...)
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if len(cronWritten) != 1 || cronWritten[0] != "0 1 * * * echo ok" {
		t.Fatalf("unexpected filtered cron lines: %#v", cronWritten)
	}
}

func TestUpdateProfileAndConfigParsing(t *testing.T) {
	td := t.TempDir()
	profile := filepath.Join(td, "profile")
	cfg := filepath.Join(td, "ShellCrash.cfg")
	if err := os.WriteFile(profile, []byte("alias old=\"sh /etc/ShellCrash/menu.sh\"\nexport CRASHDIR=\"/etc/ShellCrash\"\nPATH=/usr/bin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("my_alias=hello\nshtype=ash\nzip_type=upx\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	alias, shType, zipType := readGeneralInitConfig(cfg)
	if alias != "hello" || shType != "ash" || zipType != "upx" {
		t.Fatalf("unexpected parsed config: alias=%q shType=%q zipType=%q", alias, shType, zipType)
	}
	if err := updateProfile(profile, "/data/ShellCrash", alias, shType); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	txt := string(b)
	if strings.Count(txt, "menu.sh") != 1 {
		t.Fatalf("expected one menu alias line, got: %q", txt)
	}
	if !strings.Contains(txt, "alias hello=\"ash /data/ShellCrash/menu.sh\"") {
		t.Fatalf("missing updated alias: %q", txt)
	}
	if !strings.Contains(txt, "export CRASHDIR=\"/data/ShellCrash\"") {
		t.Fatalf("missing updated CRASHDIR export: %q", txt)
	}
}
