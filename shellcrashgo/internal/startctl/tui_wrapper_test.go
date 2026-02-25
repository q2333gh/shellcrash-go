package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusTUILayoutScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	tuiLayoutScript := filepath.Join(repoRoot, "scripts/menus/tui_layout.sh")
	if _, err := os.Stat(tuiLayoutScript); os.IsNotExist(err) {
		t.Skipf("tui_layout.sh not found at %s", tuiLayoutScript)
	}

	// Read the script content
	content, err := os.ReadFile(tuiLayoutScript)
	if err != nil {
		t.Fatalf("Failed to read tui_layout.sh: %v", err)
	}

	scriptContent := string(content)

	// Verify it's a Go-first wrapper
	if !strings.Contains(scriptContent, "Go-first wrapper") {
		t.Error("tui_layout.sh should be marked as a Go-first wrapper")
	}

	// Verify it dispatches to tuictl binary
	if !strings.Contains(scriptContent, "tuictl") {
		t.Error("tui_layout.sh should dispatch to tuictl binary")
	}

	// Verify the core functions exist
	requiredFunctions := []string{
		"content_line()",
		"sub_content_line()",
		"separator_line()",
		"line_break()",
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(scriptContent, fn) {
			t.Errorf("tui_layout.sh should define %s function", fn)
		}
	}

	// Verify shell syntax is valid
	cmd := exec.Command("bash", "-n", tuiLayoutScript)
	if err := cmd.Run(); err != nil {
		t.Errorf("tui_layout.sh has invalid shell syntax: %v", err)
	}
}

func TestMenusCommonScriptDispatchesToGoBinary(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	commonScript := filepath.Join(repoRoot, "scripts/menus/common.sh")
	if _, err := os.Stat(commonScript); os.IsNotExist(err) {
		t.Skipf("common.sh not found at %s", commonScript)
	}

	// Read the script content
	content, err := os.ReadFile(commonScript)
	if err != nil {
		t.Fatalf("Failed to read common.sh: %v", err)
	}

	scriptContent := string(content)

	// Verify it's a Go-first wrapper
	if !strings.Contains(scriptContent, "Go-first wrapper") {
		t.Error("common.sh should be marked as a Go-first wrapper")
	}

	// Verify it dispatches to tuictl binary
	if !strings.Contains(scriptContent, "tuictl") {
		t.Error("common.sh should dispatch to tuictl binary")
	}

	// Verify the core functions exist
	requiredFunctions := []string{
		"msg_alert()",
		"comp_box()",
		"top_box()",
		"btm_box()",
		"list_box()",
		"common_success()",
		"common_failed()",
		"common_back()",
		"errornum()",
		"error_letter()",
		"error_input()",
		"cancel_back()",
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(scriptContent, fn) {
			t.Errorf("common.sh should define %s function", fn)
		}
	}

	// Verify shell syntax is valid
	cmd := exec.Command("bash", "-n", commonScript)
	if err := cmd.Run(); err != nil {
		t.Errorf("common.sh has invalid shell syntax: %v", err)
	}
}
