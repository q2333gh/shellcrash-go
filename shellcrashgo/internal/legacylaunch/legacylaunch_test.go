package legacylaunch

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestRunRequiresCommand(t *testing.T) {
	err := Run(Options{Name: "shellcrash", TmpDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "command is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRequiresName(t *testing.T) {
	err := Run(Options{Command: "echo ok", TmpDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStartsDetachedAndWritesPID(t *testing.T) {
	origLookPath := lookPath
	origReadFile := readFile
	lookPath = func(file string) (string, error) {
		if file == "su" {
			return "", &os.PathError{Op: "lookpath", Path: file, Err: os.ErrNotExist}
		}
		return origLookPath(file)
	}
	readFile = func(path string) ([]byte, error) {
		if path == "/etc/passwd" {
			return []byte(""), nil
		}
		return origReadFile(path)
	}
	t.Cleanup(func() {
		lookPath = origLookPath
		readFile = origReadFile
	})

	tmpDir := t.TempDir()
	if err := Run(Options{Command: "sleep 2", Name: "legacy", TmpDir: tmpDir}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	pidPath := filepath.Join(tmpDir, "legacy.pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("read pid failed: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid pid content: %q", string(data))
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("expected process alive, pid=%d err=%v", pid, err)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
