package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibsCoreToolsScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	coreToolsScript := filepath.Join(repoRoot, "scripts/libs/core_tools.sh")
	if _, err := os.Stat(coreToolsScript); os.IsNotExist(err) {
		t.Skipf("core_tools.sh not found at %s", coreToolsScript)
	}

	// Read the script content
	content, err := os.ReadFile(coreToolsScript)
	if err != nil {
		t.Fatalf("Failed to read core_tools.sh: %v", err)
	}

	scriptContent := string(content)

	// Verify it's a Go-first wrapper
	if !strings.Contains(scriptContent, "Go-first wrapper") {
		t.Error("core_tools.sh should be marked as a Go-first wrapper")
	}

	// Verify it dispatches to utilsctl binary
	if !strings.Contains(scriptContent, "utilsctl") {
		t.Error("core_tools.sh should dispatch to utilsctl binary")
	}

	// Verify the core functions exist
	requiredFunctions := []string{
		"core_unzip()",
		"core_find()",
		"core_check()",
		"core_webget()",
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(scriptContent, fn) {
			t.Errorf("core_tools.sh should define %s function", fn)
		}
	}

	// Verify it uses utilsctl_run helper
	if !strings.Contains(scriptContent, "utilsctl_run") {
		t.Error("core_tools.sh should use utilsctl_run helper function")
	}

	// Verify it dispatches core commands to Go
	coreCommands := []string{
		"core-unzip",
		"core-find",
		"core-check",
		"core-target",
	}
	for _, cmd := range coreCommands {
		if !strings.Contains(scriptContent, cmd) {
			t.Errorf("core_tools.sh should dispatch to utilsctl %s command", cmd)
		}
	}
}
