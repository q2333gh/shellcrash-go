package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CoreUnzip extracts a core binary from various archive formats
// $1: source file path (can be .tar.gz, .gz, .upx, or raw binary)
// $2: target filename (e.g., "CrashCore" or "core_new")
// tmpDir: temporary directory for extraction
// binDir: binary directory (used for small flash mode check)
func CoreUnzip(sourceFile, targetName, tmpDir, binDir string) error {
	targetPath := filepath.Join(tmpDir, targetName)

	if strings.HasSuffix(sourceFile, ".tar.gz") {
		// Small flash mode: prevent space exhaustion
		if binDir == tmpDir {
			os.RemoveAll(filepath.Join(tmpDir, "CrashCore"))
		}

		// Create temporary extraction directory
		coreTmpDir := filepath.Join(tmpDir, "core_tmp")
		if err := os.MkdirAll(coreTmpDir, 0755); err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(coreTmpDir)

		// Extract tar.gz
		if err := extractTarGz(sourceFile, coreTmpDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz: %w", err)
		}

		// Find core binary (files > 2000 bytes matching core patterns)
		corePattern := regexp.MustCompile(`(?i)(CrashCore|sing|meta|mihomo|clash|pre)`)
		found := false
		err := filepath.Walk(coreTmpDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && info.Size() > 2000 {
				basename := filepath.Base(path)
				if corePattern.MatchString(basename) {
					if err := os.Rename(path, targetPath); err != nil {
						return err
					}
					found = true
					return filepath.SkipDir
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to find core binary: %w", err)
		}
		if !found {
			return fmt.Errorf("no core binary found in archive")
		}

	} else if strings.HasSuffix(sourceFile, ".gz") {
		// Extract .gz file
		if err := extractGz(sourceFile, targetPath); err != nil {
			return fmt.Errorf("failed to extract gz: %w", err)
		}

	} else if strings.HasSuffix(sourceFile, ".upx") {
		// Create symlink for .upx files
		if err := os.Symlink(sourceFile, targetPath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}

	} else {
		// Move raw binary
		if err := os.Rename(sourceFile, targetPath); err != nil {
			return fmt.Errorf("failed to move binary: %w", err)
		}
	}

	// Make executable
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	return nil
}

// extractTarGz extracts a tar.gz archive to the target directory
func extractTarGz(sourceFile, targetDir string) error {
	f, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

// extractGz extracts a .gz file to the target path
func extractGz(sourceFile, targetPath string) error {
	f, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	outFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, gzr)
	return err
}

// CoreFind locates and extracts the core binary if not already present
// Searches for CrashCore.* archives in BINDIR and extracts to TMPDIR
func CoreFind(tmpDir, binDir, crashDir string) error {
	corePath := filepath.Join(tmpDir, "CrashCore")
	if _, err := os.Stat(corePath); err == nil {
		// Core already exists
		return nil
	}

	// Move archives from CRASHDIR to BINDIR if different
	if crashDir != binDir {
		pattern := filepath.Join(crashDir, "CrashCore.*")
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() && info.Size() > 2000 {
				target := filepath.Join(binDir, filepath.Base(match))
				os.Rename(match, target)
			}
		}
	}

	// Find core archive in BINDIR
	pattern := filepath.Join(binDir, "CrashCore.*")
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && !info.IsDir() && info.Size() > 2000 {
			return CoreUnzip(match, "CrashCore", tmpDir, binDir)
		}
	}

	return fmt.Errorf("no core archive found")
}

// CoreCheckResult contains the result of core verification
type CoreCheckResult struct {
	Version string
	Command string
	IsValid bool
}

// CoreCheck verifies a core binary and determines its type and version
// Returns version, command template, and validity
func CoreCheck(archivePath, tmpDir, binDir, crashDir, crashCore string, stopFunc func() error) (*CoreCheckResult, error) {
	// Stop running core to prevent memory exhaustion
	if stopFunc != nil {
		if err := stopFunc(); err != nil {
			// Log but don't fail
		}
	}

	// Extract to core_new
	if err := CoreUnzip(archivePath, "core_new", tmpDir, binDir); err != nil {
		return nil, fmt.Errorf("failed to unzip core: %w", err)
	}

	coreNewPath := filepath.Join(tmpDir, "core_new")
	result := &CoreCheckResult{}

	// Check if it's a sing-box core
	isSingbox := strings.Contains(crashCore, "singbox")

	// Test the binary
	if isSingbox {
		// Check for sing-box
		cmd := exec.Command(coreNewPath, "-h")
		output, _ := cmd.CombinedOutput()
		if strings.Contains(string(output), "sing-box") {
			// Get version
			cmd = exec.Command(coreNewPath, "version")
			output, _ = cmd.Output()
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "version") {
					fields := strings.Fields(line)
					if len(fields) >= 3 {
						result.Version = fields[2]
						break
					}
				}
			}
			result.Command = `"$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons"`
			result.IsValid = true
		}
	} else {
		// Check for Clash
		cmd := exec.Command(coreNewPath, "-h")
		output, _ := cmd.CombinedOutput()
		if strings.Contains(string(output), "-t") {
			// Get version
			cmd = exec.Command(coreNewPath, "-v")
			output, _ = cmd.Output()
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				// Extract version: remove " linux.*" and get last field
				version := strings.TrimSpace(lines[0])
				version = regexp.MustCompile(` linux.*`).ReplaceAllString(version, "")
				fields := strings.Fields(version)
				if len(fields) > 0 {
					result.Version = fields[len(fields)-1]
				}
			}
			result.Command = `"$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"`
			result.IsValid = true
		}
	}

	if !result.IsValid || result.Version == "" {
		os.Remove(archivePath)
		os.Remove(coreNewPath)
		return nil, fmt.Errorf("invalid core binary")
	}

	return result, nil
}

// CoreInstall installs a verified core binary
// Should be called after CoreCheck succeeds
func CoreInstall(archivePath, tmpDir, binDir, zipType string, result *CoreCheckResult) error {
	coreNewPath := filepath.Join(tmpDir, "core_new")
	corePath := filepath.Join(tmpDir, "CrashCore")

	// Remove old archives
	os.Remove(filepath.Join(binDir, "CrashCore.tar.gz"))
	os.Remove(filepath.Join(binDir, "CrashCore.gz"))
	os.Remove(filepath.Join(binDir, "CrashCore.upx"))

	// Install new archive
	if zipType == "" {
		// Default: compress with gzip
		inFile, err := os.Open(coreNewPath)
		if err != nil {
			return err
		}
		defer inFile.Close()

		outPath := filepath.Join(binDir, "CrashCore.gz")
		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		gzw := gzip.NewWriter(outFile)
		defer gzw.Close()

		if _, err := io.Copy(gzw, inFile); err != nil {
			return err
		}
	} else {
		// Move archive with specified type
		target := filepath.Join(binDir, "CrashCore."+zipType)
		if err := os.Rename(archivePath, target); err != nil {
			return err
		}
	}

	// Install core binary
	if zipType == "upx" {
		os.Remove(archivePath)
		os.Remove(coreNewPath)
		upxPath := filepath.Join(tmpDir, "CrashCore.upx")
		if err := os.Symlink(upxPath, corePath); err != nil {
			return err
		}
	} else {
		if err := os.Rename(coreNewPath, corePath); err != nil {
			return err
		}
	}

	return nil
}

// GetCoreTarget determines the target type (clash/singbox) and format (yaml/json)
func GetCoreTarget(crashCore string) (target, format string) {
	if strings.Contains(crashCore, "singbox") {
		return "singbox", "json"
	}
	return "clash", "yaml"
}
