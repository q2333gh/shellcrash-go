package watchdog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func pidAndLockPaths(target string) (string, string) {
	tmp := "/tmp/ShellCrash"
	return filepath.Join(tmp, target+".pid"), filepath.Join(tmp, "start_"+target+".lock")
}

func cleanupPaths(target string) {
	pid, lock := pidAndLockPaths(target)
	_ = os.Remove(pid)
	_ = os.Remove(lock)
}

func TestRunBlocksWhenStartErrorWithoutMarker(t *testing.T) {
	target := "bot_tg"
	cleanupPaths(target)
	t.Cleanup(func() { cleanupPaths(target) })

	crashDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(crashDir, ".start_error"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}

	started := false
	err := Run(Options{CrashDir: crashDir, Target: target}, Deps{
		StartShellCrash: func(string) error { started = true; return nil },
		StartBotTG:      func(string, string, string) error { started = true; return nil },
	})
	if err == nil {
		t.Fatal("expected blocked startup error")
	}
	if started {
		t.Fatal("watchdog should not start services when blocked")
	}
}

func TestRunSkipsWhenLockExists(t *testing.T) {
	target := "bot_tg"
	cleanupPaths(target)
	t.Cleanup(func() { cleanupPaths(target) })

	_, lock := pidAndLockPaths(target)
	if err := os.MkdirAll("/tmp/ShellCrash", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}

	started := false
	err := Run(Options{CrashDir: t.TempDir(), Target: target}, Deps{
		StartBotTG: func(string, string, string) error { started = true; return nil },
	})
	if err != nil {
		t.Fatalf("expected lock skip, got %v", err)
	}
	if started {
		t.Fatal("watchdog should not start while locked")
	}
}

func TestRunSkipsWhenPIDAlive(t *testing.T) {
	target := "bot_tg"
	cleanupPaths(target)
	t.Cleanup(func() { cleanupPaths(target) })

	pidFile, _ := pidAndLockPaths(target)
	if err := os.MkdirAll("/tmp/ShellCrash", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte("12345\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	started := false
	err := Run(Options{CrashDir: t.TempDir(), Target: target}, Deps{
		IsProcessAlive: func(pid int) bool { return pid == 12345 },
		StartBotTG:     func(string, string, string) error { started = true; return nil },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if started {
		t.Fatal("watchdog should skip start when pid is alive")
	}
}

func TestRunStartsWhenPIDInvalid(t *testing.T) {
	target := "bot_tg"
	cleanupPaths(target)
	t.Cleanup(func() { cleanupPaths(target) })

	pidFile, _ := pidAndLockPaths(target)
	if err := os.MkdirAll("/tmp/ShellCrash", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	started := false
	err := Run(Options{CrashDir: t.TempDir(), Target: target}, Deps{
		StartBotTG: func(_, _ string, gotPID string) error {
			started = true
			if gotPID != pidFile {
				t.Fatalf("unexpected pid file: %s", gotPID)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !started {
		t.Fatal("expected watchdog to start target")
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("expected invalid pid file removed, err=%v", err)
	}
}

func TestStartDetachedDirectWritesPID(t *testing.T) {
	bin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not available")
	}
	pidFile := filepath.Join(t.TempDir(), "bot.pid")
	if err := startDetachedDirect(bin, []string{"2"}, pidFile); err != nil {
		t.Fatalf("startDetachedDirect failed: %v", err)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read pid file failed: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid pid written: %q", string(data))
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("expected process alive, pid=%d err=%v", pid, err)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

func TestRunRejectsUnsupportedTarget(t *testing.T) {
	err := Run(Options{CrashDir: t.TempDir(), Target: "ddns"}, Deps{})
	if err == nil || !strings.Contains(err.Error(), "unsupported watchdog target") {
		t.Fatalf("expected unsupported target error, got: %v", err)
	}
}
