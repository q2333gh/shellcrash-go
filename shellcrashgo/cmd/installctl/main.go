package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"archive/tar"

	"shellcrash/internal/initctl"
	"shellcrash/internal/installctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	tmpDir := flag.String("tmpdir", os.Getenv("TMPDIR"), "temporary directory for downloads")
	fsRoot := flag.String("fsroot", "", "filesystem root override for testing")
	url := flag.String("url", "", "base URL for ShellCrash.tar.gz and version")
	flag.Parse()

	opts := installctl.Options{
		CrashDir: *crashDir,
		TmpDir:   *tmpDir,
		FSRoot:   *fsRoot,
		URL:      *url,
		Out:      os.Stdout,
		In:       os.Stdin,
	}

	deps := installctl.Deps{
		DownloadVersion: downloadVersionHTTP,
		DownloadArchive: downloadFileHTTP,
		ExtractTarGz:    extractTarGz,
		RunInit:         initctl.Run,
	}

	if err := installctl.Run(opts, deps); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func downloadVersionHTTP(url string) (string, error) {
	resp, err := http.Get(url) // #nosec G107 - URL is controlled by caller/flags
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func downloadFileHTTP(url, destPath string) error {
	resp, err := http.Get(url) // #nosec G107 - URL is controlled by caller/flags
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
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
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			// Ignore other types for now (symlinks, etc.)
		}
	}
	return nil
}

