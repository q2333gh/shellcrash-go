package uninstallctl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMenuCancelDoesNotUninstall(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "etc", "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("0\n")
	var out bytes.Buffer
	called := false
	err := RunMenu(Options{CrashDir: crashDir, FSRoot: root}, Deps{
		StartAction: func(string, string, []string) error {
			called = true
			return nil
		},
		RunCommand: func(string, ...string) error { return nil },
	}, input, &out)
	if err != nil {
		t.Fatalf("run menu error: %v", err)
	}
	if called {
		t.Fatalf("expected no uninstall actions when canceled")
	}
	if _, statErr := os.Stat(crashDir); statErr != nil {
		t.Fatalf("expected crashdir retained on cancel: %v", statErr)
	}
}

func TestRunMenuConfirmKeepConfig(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "etc", "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), []byte("x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("1\n1\n")
	var out bytes.Buffer
	err := RunMenu(Options{CrashDir: crashDir, FSRoot: root}, Deps{
		StartAction: func(string, string, []string) error { return nil },
		RunCommand:  func(string, ...string) error { return nil },
	}, input, &out)
	if err != nil {
		t.Fatalf("run menu error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(crashDir, "configs")); statErr != nil {
		t.Fatalf("expected configs retained: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(crashDir, "bin")); !os.IsNotExist(statErr) {
		t.Fatalf("expected runtime content removed, statErr=%v", statErr)
	}
}
