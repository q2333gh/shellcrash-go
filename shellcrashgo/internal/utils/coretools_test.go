package utils

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestGetCoreTarget(t *testing.T) {
	tests := []struct {
		crashCore string
		target    string
		format    string
	}{
		{"singbox", "singbox", "json"},
		{"clash", "clash", "yaml"},
		{"meta", "clash", "yaml"},
		{"mihomo", "clash", "yaml"},
	}

	for _, tt := range tests {
		target, format := GetCoreTarget(tt.crashCore)
		if target != tt.target {
			t.Errorf("GetCoreTarget(%q) target = %q, want %q", tt.crashCore, target, tt.target)
		}
		if format != tt.format {
			t.Errorf("GetCoreTarget(%q) format = %q, want %q", tt.crashCore, format, tt.format)
		}
	}
}

func TestCoreUnzipGz(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := tmpDir

	// Create a test .gz file
	testContent := []byte("test binary content")
	gzPath := filepath.Join(tmpDir, "test.gz")
	f, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	gzw := gzip.NewWriter(f)
	gzw.Write(testContent)
	gzw.Close()
	f.Close()

	// Extract
	err = CoreUnzip(gzPath, "extracted", tmpDir, binDir)
	if err != nil {
		t.Fatalf("CoreUnzip failed: %v", err)
	}

	// Verify
	extractedPath := filepath.Join(tmpDir, "extracted")
	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("Extracted content = %q, want %q", content, testContent)
	}

	// Check executable permission
	info, err := os.Stat(extractedPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("Extracted file is not executable")
	}
}

func TestCoreUnzipRawBinary(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := tmpDir

	// Create a test raw binary
	testContent := []byte("raw binary content")
	rawPath := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(rawPath, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Extract (should just move)
	err := CoreUnzip(rawPath, "extracted", tmpDir, binDir)
	if err != nil {
		t.Fatalf("CoreUnzip failed: %v", err)
	}

	// Verify
	extractedPath := filepath.Join(tmpDir, "extracted")
	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("Extracted content = %q, want %q", content, testContent)
	}

	// Original should be gone (moved)
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Error("Original file still exists after move")
	}
}

func TestCoreFindNoCore(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := t.TempDir()
	crashDir := t.TempDir()

	err := CoreFind(tmpDir, binDir, crashDir)
	if err == nil {
		t.Error("CoreFind should fail when no core archive exists")
	}
}

func TestCoreFindExistingCore(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := tmpDir
	crashDir := tmpDir

	// Create existing core
	corePath := filepath.Join(tmpDir, "CrashCore")
	if err := os.WriteFile(corePath, []byte("existing"), 0755); err != nil {
		t.Fatal(err)
	}

	// Should not fail and not overwrite
	err := CoreFind(tmpDir, binDir, crashDir)
	if err != nil {
		t.Errorf("CoreFind failed: %v", err)
	}

	content, _ := os.ReadFile(corePath)
	if string(content) != "existing" {
		t.Error("CoreFind overwrote existing core")
	}
}

func TestCoreInstallDefault(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := t.TempDir()

	// Create core_new
	coreNewPath := filepath.Join(tmpDir, "core_new")
	testContent := []byte("new core binary")
	if err := os.WriteFile(coreNewPath, testContent, 0755); err != nil {
		t.Fatal(err)
	}

	result := &CoreCheckResult{
		Version: "1.0.0",
		Command: "test command",
		IsValid: true,
	}

	// Install with default compression
	err := CoreInstall("", tmpDir, binDir, "", result)
	if err != nil {
		t.Fatalf("CoreInstall failed: %v", err)
	}

	// Verify CrashCore.gz exists
	gzPath := filepath.Join(binDir, "CrashCore.gz")
	if _, err := os.Stat(gzPath); os.IsNotExist(err) {
		t.Error("CrashCore.gz not created")
	}

	// Verify CrashCore exists
	corePath := filepath.Join(tmpDir, "CrashCore")
	if _, err := os.Stat(corePath); os.IsNotExist(err) {
		t.Error("CrashCore not created")
	}
}
