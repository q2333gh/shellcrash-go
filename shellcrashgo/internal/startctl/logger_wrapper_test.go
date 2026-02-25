package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibsLoggerScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	loggerScript := filepath.Join(repoRoot, "scripts/libs/logger.sh")
	if _, err := os.Stat(loggerScript); os.IsNotExist(err) {
		t.Skipf("logger.sh not found at %s", loggerScript)
	}

	// Read the script content
	content, err := os.ReadFile(loggerScript)
	if err != nil {
		t.Fatalf("Failed to read logger.sh: %v", err)
	}

	scriptContent := string(content)

	// Verify it's a Go-first wrapper
	if !strings.Contains(scriptContent, "Go-first wrapper") {
		t.Error("logger.sh should be marked as a Go-first wrapper")
	}

	// Verify it dispatches to loggerctl binary
	if !strings.Contains(scriptContent, "loggerctl") {
		t.Error("logger.sh should dispatch to loggerctl binary")
	}

	// Verify the logger function exists
	if !strings.Contains(scriptContent, "logger()") {
		t.Error("logger.sh should define logger() function")
	}

	// Verify shell syntax is valid
	cmd := exec.Command("bash", "-n", loggerScript)
	if err := cmd.Run(); err != nil {
		t.Errorf("logger.sh has invalid shell syntax: %v", err)
	}
}
