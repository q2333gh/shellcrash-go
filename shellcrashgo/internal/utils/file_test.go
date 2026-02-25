package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	content1 := []byte("hello world")
	content2 := []byte("hello world")
	content3 := []byte("different content")

	if err := os.WriteFile(file1, content1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, content2, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, content3, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path1   string
		path2   string
		want    bool
		wantErr bool
	}{
		{
			name:    "identical files",
			path1:   file1,
			path2:   file2,
			want:    true,
			wantErr: false,
		},
		{
			name:    "different files",
			path1:   file1,
			path2:   file3,
			want:    false,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path1:   file1,
			path2:   filepath.Join(tmpDir, "nonexistent.txt"),
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareFiles(tt.path1, tt.path2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CompareFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareFilesLarge(t *testing.T) {
	tmpDir := t.TempDir()

	// Create large test files (larger than chunk size)
	file1 := filepath.Join(tmpDir, "large1.txt")
	file2 := filepath.Join(tmpDir, "large2.txt")

	// Create 20KB of data (larger than 8KB chunk size)
	largeContent := make([]byte, 20*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	if err := os.WriteFile(file1, largeContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	same, err := CompareFiles(file1, file2)
	if err != nil {
		t.Fatalf("CompareFiles() error = %v", err)
	}
	if !same {
		t.Errorf("CompareFiles() = false, want true for identical large files")
	}
}
