package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWebGet(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer ts.Close()

	// Create temp file
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "test.txt")

	// Test download
	err := WebGet(localPath, ts.URL, nil)
	if err != nil {
		t.Fatalf("WebGet failed: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got '%s'", string(content))
	}
}

func TestWebGetWithOptions(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check user agent
		ua := r.Header.Get("User-Agent")
		if ua != "CustomAgent/1.0" {
			t.Errorf("Expected User-Agent 'CustomAgent/1.0', got '%s'", ua)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "test.txt")

	opts := &WebGetOptions{
		UserAgent: "CustomAgent/1.0",
		Timeout:   10,
	}

	err := WebGet(localPath, ts.URL, opts)
	if err != nil {
		t.Fatalf("WebGet with options failed: %v", err)
	}
}

func TestWebGetNoRedirect(t *testing.T) {
	// Create test server with redirect
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("target content"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "test.txt")

	opts := &WebGetOptions{
		NoRedirect: true,
	}

	// Should fail because redirect is disabled
	err := WebGet(localPath, ts.URL+"/redirect", opts)
	if err == nil {
		t.Error("Expected error with NoRedirect, got nil")
	}
}

func TestRewriteGitHubURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		proxyActive bool
		expected    string
	}{
		{
			name:        "jsdelivr to raw with proxy",
			input:       "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master/test.sh",
			proxyActive: true,
			expected:    "https://raw.githubusercontent.com/juewuy/ShellCrash/master/test.sh",
		},
		{
			name:        "raw to jsdelivr without proxy",
			input:       "https://raw.githubusercontent.com/juewuy/ShellCrash/master/test.sh",
			proxyActive: false,
			expected:    "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master/test.sh",
		},
		{
			name:        "gh.jwsc.eu.org to raw with proxy",
			input:       "https://gh.jwsc.eu.org/master/test.sh",
			proxyActive: true,
			expected:    "https://raw.githubusercontent.com/juewuy/ShellCrash/master/test.sh",
		},
		{
			name:        "no change for other URLs",
			input:       "https://example.com/test.sh",
			proxyActive: true,
			expected:    "https://example.com/test.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteGitHubURL(tt.input, tt.proxyActive)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetBin(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("binary content"))
	}))
	defer ts.Close()

	// Create temp directories
	tmpDir := t.TempDir()
	crashDir := filepath.Join(tmpDir, "crash")
	os.MkdirAll(filepath.Join(crashDir, "configs"), 0755)

	localPath := filepath.Join(tmpDir, "test.bin")

	// Test with default update URL
	opts := &GetBinOptions{
		UpdateURL: ts.URL,
	}

	err := GetBin(localPath, "test/file.bin", crashDir, opts)
	if err != nil {
		t.Fatalf("GetBin failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Error("Downloaded file does not exist")
	}
}

func TestGetBinWithServersList(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("binary content"))
	}))
	defer ts.Close()

	// Create temp directories
	tmpDir := t.TempDir()
	crashDir := filepath.Join(tmpDir, "crash")
	os.MkdirAll(filepath.Join(crashDir, "configs"), 0755)

	// Create servers.list - use ID 102 (not jsdelivr) to avoid @ syntax
	serversContent := "102 TestServer " + ts.URL + "\n"
	os.WriteFile(filepath.Join(crashDir, "configs", "servers.list"), []byte(serversContent), 0644)

	localPath := filepath.Join(tmpDir, "test.bin")

	opts := &GetBinOptions{
		URLId: "102",
	}

	err := GetBin(localPath, "bin/test.bin", crashDir, opts)
	if err != nil {
		t.Fatalf("GetBin with servers.list failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Error("Downloaded file does not exist")
	}
}
