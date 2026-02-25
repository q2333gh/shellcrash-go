package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerBasicLogging(t *testing.T) {
	tmpDir := t.TempDir()
	config := &Config{}
	l := New(tmpDir, config)

	err := l.Log("test message", "", false, false)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	logFile := filepath.Join(tmpDir, "ShellCrash.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test message") {
		t.Errorf("Log file does not contain expected message. Got: %s", content)
	}
}

func TestLoggerOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	config := &Config{}
	l := New(tmpDir, config)

	l.Log("test message 1", "", false, false)
	l.Log("test message 2", "", false, false)
	l.Log("test message 1", "", false, true)

	logFile := filepath.Join(tmpDir, "ShellCrash.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")

	count := 0
	for _, line := range lines {
		if strings.Contains(line, "test message 1") {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 occurrence of 'test message 1', got %d. Content: %s", count, content)
	}
}

func TestLoggerTrim(t *testing.T) {
	tmpDir := t.TempDir()
	config := &Config{}
	l := New(tmpDir, config)

	for i := 0; i < 250; i++ {
		l.Log("test message", "", false, false)
	}

	logFile := filepath.Join(tmpDir, "ShellCrash.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > maxLogLines {
		t.Errorf("Log file has %d lines, expected <= %d", len(lines), maxLogLines)
	}
}

func TestLoggerWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	config := &Config{
		DeviceName: "test-device",
	}
	l := New(tmpDir, config)

	err := l.Log("test message", "", false, false)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	logFile := filepath.Join(tmpDir, "ShellCrash.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test message") {
		t.Errorf("Log file does not contain expected message")
	}
}

func TestNewLoggerDefaults(t *testing.T) {
	l := New("", nil)
	if l.tmpDir != "/tmp/ShellCrash" {
		t.Errorf("Expected default tmpDir '/tmp/ShellCrash', got '%s'", l.tmpDir)
	}
	if l.config == nil {
		t.Error("Expected non-nil config")
	}
}
