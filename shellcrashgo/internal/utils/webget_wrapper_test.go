package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibsWebGetScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	webGetScript := filepath.Join(repoRoot, "scripts/libs/web_get.sh")
	if _, err := os.Stat(webGetScript); os.IsNotExist(err) {
		t.Skipf("web_get.sh not found at %s", webGetScript)
	}

	// Read the script content
	content, err := os.ReadFile(webGetScript)
	if err != nil {
		t.Fatalf("Failed to read web_get.sh: %v", err)
	}

	scriptContent := string(content)

	// Verify it's a Go-first wrapper
	if !strings.Contains(scriptContent, "Go-first wrapper") {
		t.Error("web_get.sh should be marked as a Go-first wrapper")
	}

	// Verify it dispatches to webgetctl binary
	if !strings.Contains(scriptContent, "webgetctl") {
		t.Error("web_get.sh should dispatch to webgetctl binary")
	}

	// Verify the core function exists
	if !strings.Contains(scriptContent, "webget()") {
		t.Error("web_get.sh should define webget() function")
	}

	// Verify it uses webgetctl_run helper
	if !strings.Contains(scriptContent, "webgetctl_run") {
		t.Error("web_get.sh should use webgetctl_run helper function")
	}
}

func TestLibsWebGetBinScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	webGetBinScript := filepath.Join(repoRoot, "scripts/libs/web_get_bin.sh")
	if _, err := os.Stat(webGetBinScript); os.IsNotExist(err) {
		t.Skipf("web_get_bin.sh not found at %s", webGetBinScript)
	}

	// Read the script content
	content, err := os.ReadFile(webGetBinScript)
	if err != nil {
		t.Fatalf("Failed to read web_get_bin.sh: %v", err)
	}

	scriptContent := string(content)

	// Verify it's a Go-first wrapper
	if !strings.Contains(scriptContent, "Go-first wrapper") {
		t.Error("web_get_bin.sh should be marked as a Go-first wrapper")
	}

	// Verify it dispatches to webgetctl binary
	if !strings.Contains(scriptContent, "webgetctl") {
		t.Error("web_get_bin.sh should dispatch to webgetctl binary")
	}

	// Verify the core function exists
	if !strings.Contains(scriptContent, "get_bin()") {
		t.Error("web_get_bin.sh should define get_bin() function")
	}

	// Verify it uses webgetctl_run helper
	if !strings.Contains(scriptContent, "webgetctl_run") {
		t.Error("web_get_bin.sh should use webgetctl_run helper function")
	}

	// Verify it dispatches get-bin command to Go
	if !strings.Contains(scriptContent, "get-bin") {
		t.Error("web_get_bin.sh should dispatch to webgetctl get-bin command")
	}
}
