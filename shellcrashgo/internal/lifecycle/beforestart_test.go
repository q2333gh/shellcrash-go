package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBeforeStartCoreFlow(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "crash")
	binDir := filepath.Join(td, "bin")
	tmpDir := filepath.Join(td, "tmp")
	if err := os.MkdirAll(filepath.Join(crashDir, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "providers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "providers", "a.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "task", "bfstart"), []byte("echo pre\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	coreConfig := filepath.Join(crashDir, "yamls", "config.yaml")
	ensureCalled := false
	taskCalled := false
	err := BeforeStart(BeforeStartOptions{
		CrashDir:   crashDir,
		BinDir:     binDir,
		TmpDir:     tmpDir,
		CoreConfig: coreConfig,
		URL:        "https://example.com/sub",
		Host:       "192.168.1.2",
		MixPort:    "7890",
	}, BeforeStartDeps{
		EnsureCoreConfig: func() error {
			ensureCalled = true
			if err := os.MkdirAll(filepath.Dir(coreConfig), 0o755); err != nil {
				return err
			}
			return os.WriteFile(coreConfig, []byte("ok"), 0o644)
		},
		RunTaskScript: func(path string) error {
			taskCalled = strings.HasSuffix(path, "/task/bfstart")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ensureCalled {
		t.Fatalf("expected EnsureCoreConfig to be called")
	}
	if !taskCalled {
		t.Fatalf("expected bfstart task to be called")
	}
	if !fileExists(filepath.Join(binDir, "ui", "index.html")) {
		t.Fatalf("index.html should be created")
	}
	pac, err := os.ReadFile(filepath.Join(binDir, "ui", "pac"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pac), "192.168.1.2:7890") {
		t.Fatalf("unexpected pac content: %q", string(pac))
	}
	providerLink := filepath.Join(binDir, "providers", "a.yaml")
	info, err := os.Lstat(providerLink)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("provider should be symlinked")
	}
}

func TestBeforeStartRequiresConfigLinkWhenMissingCoreConfig(t *testing.T) {
	td := t.TempDir()
	err := BeforeStart(BeforeStartOptions{
		CrashDir:   td,
		BinDir:     td,
		TmpDir:     filepath.Join(td, "tmp"),
		CoreConfig: filepath.Join(td, "yamls", "config.yaml"),
	}, BeforeStartDeps{
		EnsureCoreConfig: func() error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "core config link missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}
