package startctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCoreErrorSnippet(t *testing.T) {
	logText := strings.Join([]string{
		"info: booting",
		"FATAL: config broken",
		"warn: retry",
		"error = parse failed",
	}, "\n")

	out := extractCoreErrorSnippet(logText)
	if !strings.Contains(out, "FATAL: config broken") {
		t.Fatalf("expected fatal line in snippet, got %q", out)
	}
	if !strings.Contains(out, "error = parse failed") {
		t.Fatalf("expected error line in snippet, got %q", out)
	}
}

func TestRunCoreCommandForErrorLog(t *testing.T) {
	td := t.TempDir()
	logPath := filepath.Join(td, "core_test.log")
	ctl := Controller{
		Cfg: Config{
			CrashDir: td,
			TmpDir:   td,
			BinDir:   td,
			Command:  "echo ERROR_LINE",
		},
	}

	if err := ctl.runCoreCommandForErrorLog(logPath); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "ERROR_LINE") {
		t.Fatalf("unexpected core_test.log: %q", string(b))
	}
}
