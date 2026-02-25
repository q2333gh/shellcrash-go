package tgbot

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetConfigValuesUpdatesAndAppends(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "ShellCrash.cfg")
	orig := "firewall_area=1\nredir_mod=Tun\n# keep\n"
	if err := os.WriteFile(cfg, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setConfigValues(cfg, map[string]string{
		"firewall_area": "4",
		"redir_mod_bf":  "Tun",
	}); err != nil {
		t.Fatalf("setConfigValues failed: %v", err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "firewall_area=4") {
		t.Fatalf("expected firewall_area update, got: %s", text)
	}
	if !strings.Contains(text, "redir_mod_bf=Tun") {
		t.Fatalf("expected redir_mod_bf append, got: %s", text)
	}
	if !strings.Contains(text, "# keep") {
		t.Fatalf("expected existing comment preserved, got: %s", text)
	}
}

func TestTailLinesFiltersTaskAndLimits(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "ShellCrash.log")
	content := strings.Join([]string{
		"line1",
		"任务 cron check",
		"line2",
		"",
		"line3",
	}, "\n")
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := tailLines(logFile, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 lines, got %d (%v)", len(got), got)
	}
	if got[0] != "line2" || got[1] != "line3" {
		t.Fatalf("unexpected tail lines: %v", got)
	}
}

func TestUploadStateReadWrite(t *testing.T) {
	crashDir := t.TempDir()
	mustWriteFile(t, filepath.Join(crashDir, "configs", "command.env"), "TMPDIR="+filepath.Join(crashDir, "tmp")+"\n")

	if err := writeUploadState(crashDir, uploadCfg); err != nil {
		t.Fatalf("writeUploadState failed: %v", err)
	}
	state, err := readUploadState(crashDir)
	if err != nil {
		t.Fatalf("readUploadState failed: %v", err)
	}
	if state != uploadCfg {
		t.Fatalf("unexpected state: %q", state)
	}
	clearUploadState(crashDir)
	state, err = readUploadState(crashDir)
	if err != nil {
		t.Fatalf("readUploadState after clear failed: %v", err)
	}
	if state != "" {
		t.Fatalf("expected empty state after clear, got %q", state)
	}
}

func TestProcessUploadedFileConfig(t *testing.T) {
	crashDir := t.TempDir()
	cfg := map[string]string{"crashcore": "meta"}
	src := filepath.Join(crashDir, "u.yaml")
	mustWriteBytes(t, src, []byte("port: 7890\n"), 0o644)

	msg, err := processUploadedFile(Deps{}, crashDir, cfg, uploadCfg, src, "custom.yaml")
	if err != nil {
		t.Fatalf("processUploadedFile failed: %v", err)
	}
	if !strings.Contains(msg, "已上传") {
		t.Fatalf("unexpected message: %s", msg)
	}
	dst := filepath.Join(crashDir, "yamls", "custom.yaml")
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("expected uploaded file: %v", err)
	}
	if string(b) != "port: 7890\n" {
		t.Fatalf("unexpected uploaded content: %q", string(b))
	}
}

func TestProcessUploadedFileBackup(t *testing.T) {
	crashDir := t.TempDir()
	archive := filepath.Join(crashDir, "backup.tar.gz")
	makeTarGz(t, archive, map[string]string{"ShellCrash.cfg": "crashcore=meta\n"})

	msg, err := processUploadedFile(Deps{}, crashDir, map[string]string{}, uploadBak, archive, "backup.tar.gz")
	if err != nil {
		t.Fatalf("processUploadedFile backup failed: %v", err)
	}
	if !strings.Contains(msg, "已还原") {
		t.Fatalf("unexpected message: %s", msg)
	}
	b, err := os.ReadFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("missing restored config: %v", err)
	}
	if !strings.Contains(string(b), "crashcore=meta") {
		t.Fatalf("unexpected restored content: %q", string(b))
	}
}

func TestProcessUploadedFileCoreTriggersRestart(t *testing.T) {
	crashDir := t.TempDir()
	cfgDir := filepath.Join(crashDir, "configs")
	mustWriteFile(t, filepath.Join(cfgDir, "ShellCrash.cfg"), "crashcore=meta\n")
	mustWriteFile(t, filepath.Join(cfgDir, "command.env"), "TMPDIR="+filepath.Join(crashDir, "tmp")+"\nBINDIR="+filepath.Join(crashDir, "bin")+"\n")

	src := filepath.Join(crashDir, "CrashCore.gz")
	mustWriteBytes(t, src, []byte(strings.Repeat("x", 4096)), 0o644)
	called := false
	deps := Deps{
		RunControllerAction: func(crashDir string, action string) error {
			if action != "restart" {
				t.Fatalf("unexpected action: %s", action)
			}
			called = true
			return nil
		},
	}
	msg, err := processUploadedFile(deps, crashDir, map[string]string{}, uploadCore, src, "core.gz")
	if err != nil {
		t.Fatalf("processUploadedFile core failed: %v", err)
	}
	if !called {
		t.Fatalf("expected restart action")
	}
	if !strings.Contains(msg, "重启") {
		t.Fatalf("unexpected message: %s", msg)
	}
	if _, err := os.Stat(filepath.Join(crashDir, "bin", "CrashCore.gz")); err != nil {
		t.Fatalf("expected core artifact in BINDIR: %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	mustWriteBytes(t, path, []byte(content), 0o644)
}

func mustWriteBytes(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
}

func makeTarGz(t *testing.T, out string, files map[string]string) {
	t.Helper()
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatal(err)
		}
	}
}
