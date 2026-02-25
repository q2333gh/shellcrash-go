package setbootctl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadStateDefaults(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "ShellCrash")
	if err := os.MkdirAll(filepath.Join(crashDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := LoadState(Options{CrashDir: crashDir})
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if st.StartOld != "OFF" || st.NetworkCheck != "ON" {
		t.Fatalf("unexpected defaults: %+v", st)
	}
	if st.BindDir != crashDir {
		t.Fatalf("unexpected bindir: %s", st.BindDir)
	}
}

func TestToggleConservativeModeEnablesStartOld(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("start_old=OFF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ran := make([]string, 0, 2)
	err := ToggleConservativeMode(Options{CrashDir: crashDir}, Deps{
		RunCommand: func(name string, args ...string) error {
			ran = append(ran, name+" "+strings.Join(args, " "))
			return nil
		},
	}, State{StartOld: "OFF"})
	if err != nil {
		t.Fatalf("toggle mode: %v", err)
	}
	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgOut), "start_old=ON") {
		t.Fatalf("expected start_old=ON, got: %s", string(cfgOut))
	}
	if len(ran) == 0 {
		t.Fatalf("expected stop action invocation")
	}
}

func TestSetBindDirWritesCommandEnv(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	target := filepath.Join(root, "data")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetBindDir(Options{CrashDir: crashDir}, target); err != nil {
		t.Fatalf("set bindir: %v", err)
	}
	envOut, err := os.ReadFile(filepath.Join(cfgDir, "command.env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envOut), "BINDIR="+target) {
		t.Fatalf("missing BINDIR in command.env: %s", string(envOut))
	}
}

func TestRunMenuSetStartDelay(t *testing.T) {
	root := t.TempDir()
	crashDir := filepath.Join(root, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("start_old=OFF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("TMPDIR=/tmp/ShellCrash\nBINDIR="+crashDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	in := strings.NewReader("3\n45\n0\n")
	var out bytes.Buffer
	if err := RunMenu(Options{CrashDir: crashDir}, Deps{}, in, &out); err != nil {
		t.Fatalf("run menu: %v", err)
	}
	cfgOut, err := os.ReadFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgOut), "start_delay=45") {
		t.Fatalf("expected start_delay=45, got: %s", string(cfgOut))
	}
}
