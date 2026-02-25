package installctl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"shellcrash/internal/initctl"
)

func TestRun_UsesDepsAndPaths(t *testing.T) {
	tmp := t.TempDir()
	crashDir := filepath.Join(tmp, "etc", "ShellCrash")
	tmpDir := filepath.Join(tmp, "tmp", "ShellCrash")

	var calls []string
	deps := Deps{
		DownloadVersion: func(url string) (string, error) {
			calls = append(calls, "version:"+url)
			return "1.2.3", nil
		},
		DownloadArchive: func(url, destPath string) error {
			calls = append(calls, "archive:"+url+"->"+destPath)
			return nil
		},
		ExtractTarGz: func(archivePath, destDir string) error {
			calls = append(calls, "extract:"+archivePath+"->"+destDir)
			return nil
		},
		RunInit: func(opts initctl.Options) error {
			calls = append(calls, "init:"+opts.CrashDir+"|"+opts.TmpDir+"|"+opts.FSRoot)
			return nil
		},
	}

	var out bytes.Buffer
	opts := Options{
		CrashDir: crashDir,
		TmpDir:   tmpDir,
		FSRoot:   "/",
		URL:      "https://example.com/gh/juewuy/ShellCrash@master",
		Out:      &out,
	}

	if err := Run(opts, deps); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(calls) != 4 {
		t.Fatalf("expected 4 dep calls, got %d: %v", len(calls), calls)
	}
	wantPrefix := "version:https://example.com/gh/juewuy/ShellCrash@master/version"
	if calls[0] != wantPrefix {
		t.Errorf("first call = %q, want %q", calls[0], wantPrefix)
	}
	if !strings.HasPrefix(calls[1], "archive:https://example.com/gh/juewuy/ShellCrash@master/ShellCrash.tar.gz->") {
		t.Errorf("archive call = %q, unexpected", calls[1])
	}
	if !strings.HasPrefix(calls[2], "extract:") {
		t.Errorf("extract call = %q, unexpected", calls[2])
	}
	if !strings.HasPrefix(calls[3], "init:"+crashDir+"|"+tmpDir+"|/") {
		t.Errorf("init call = %q, want prefix %q", calls[3], "init:"+crashDir+"|"+tmpDir+"|/")
	}

	output := out.String()
	if !strings.Contains(output, "ShellCrash Go installer completed successfully.") {
		t.Errorf("output %q does not contain success message", output)
	}
}

