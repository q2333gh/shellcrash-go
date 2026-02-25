package startctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCronsetUpdatesSystemCronAndPersistsTaskFile(t *testing.T) {
	crashDir := t.TempDir()
	store := filepath.Join(t.TempDir(), "crontab.txt")
	if err := os.WriteFile(store, []byte(
		"0 0 * * * echo keep\n"+
			"1 0 * * * echo old #ShellCrash初始化\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
	restoreEnv := installFakeCrontab(t, store)
	defer restoreEnv()

	oldPersistent := cronsetHasPersistentStore
	cronsetHasPersistentStore = func() bool { return true }
	defer func() { cronsetHasPersistentStore = oldPersistent }()

	c := Controller{Cfg: Config{CrashDir: crashDir}}
	if err := c.cronset([]string{"ShellCrash初始化", "2 0 * * * echo new #ShellCrash初始化"}); err != nil {
		t.Fatal(err)
	}

	gotStore, err := os.ReadFile(store)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotStore)
	if strings.Contains(got, "echo old") {
		t.Fatalf("old keyword entry should be removed, got: %s", got)
	}
	if !strings.Contains(got, "echo keep") || !strings.Contains(got, "echo new") {
		t.Fatalf("expected keep+new entries in crontab, got: %s", got)
	}

	taskCron := filepath.Join(crashDir, "task", "cron")
	taskData, err := os.ReadFile(taskCron)
	if err != nil {
		t.Fatalf("expected persisted task cron file: %v", err)
	}
	if strings.TrimSpace(string(taskData)) != strings.TrimSpace(got) {
		t.Fatalf("task cron content mismatch: task=%q store=%q", string(taskData), got)
	}
}

func TestCronsetRemovesTaskCronFileWhenNotPersistent(t *testing.T) {
	crashDir := t.TempDir()
	store := filepath.Join(t.TempDir(), "crontab.txt")
	if err := os.WriteFile(store, []byte("1 0 * * * echo old #TG_BOT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restoreEnv := installFakeCrontab(t, store)
	defer restoreEnv()

	taskCron := filepath.Join(crashDir, "task", "cron")
	if err := os.MkdirAll(filepath.Dir(taskCron), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taskCron, []byte("legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldPersistent := cronsetHasPersistentStore
	cronsetHasPersistentStore = func() bool { return false }
	defer func() { cronsetHasPersistentStore = oldPersistent }()

	c := Controller{Cfg: Config{CrashDir: crashDir}}
	if err := c.cronset([]string{"TG_BOT"}); err != nil {
		t.Fatal(err)
	}

	gotStore, err := os.ReadFile(store)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(gotStore), "TG_BOT") {
		t.Fatalf("expected TG_BOT keyword removed, got: %s", string(gotStore))
	}
	if _, err := os.Stat(taskCron); !os.IsNotExist(err) {
		t.Fatalf("expected task cron removed, stat err=%v", err)
	}
}

func installFakeCrontab(t *testing.T, storePath string) func() {
	t.Helper()
	fakeBin := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-l\" ]; then\n" +
		"  [ -f \"$CRON_STORE\" ] && cat \"$CRON_STORE\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"cat \"$1\" > \"$CRON_STORE\"\n"
	crontabPath := filepath.Join(fakeBin, "crontab")
	if err := os.WriteFile(crontabPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	oldStore := os.Getenv("CRON_STORE")
	if err := os.Setenv("PATH", fakeBin+":"+oldPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("CRON_STORE", storePath); err != nil {
		t.Fatal(err)
	}

	return func() {
		_ = os.Setenv("PATH", oldPath)
		_ = os.Setenv("CRON_STORE", oldStore)
	}
}

