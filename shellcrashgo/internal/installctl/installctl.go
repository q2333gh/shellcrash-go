package installctl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"shellcrash/internal/initctl"
)

// Options controls the high-level install behaviour.
// It is intentionally minimal for now and focuses on a pure-Go
// "download → extract → init" flow.
type Options struct {
	CrashDir string
	TmpDir   string
	FSRoot   string
	URL      string

	In  io.Reader
	Out io.Writer
}

// Deps bundles side-effecting operations so tests can stub them out.
type Deps struct {
	DownloadVersion func(url string) (string, error)
	DownloadArchive func(url, destPath string) error
	ExtractTarGz    func(archivePath, destDir string) error
	RunInit         func(opts initctl.Options) error
}

// Run performs a non-interactive installation using only Go code.
// It does not yet implement all of the legacy shell installer features
// (like alias/profile setup or sysType-specific flows), but it moves the
// core "download + extract + init" path into Go.
func Run(opts Options, deps Deps) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	tmpDir := strings.TrimSpace(opts.TmpDir)
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	fsRoot := strings.TrimSpace(opts.FSRoot)
	if fsRoot == "" {
		fsRoot = "/"
	}
	baseURL := strings.TrimSpace(opts.URL)
	if baseURL == "" {
		baseURL = "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master"
	}

	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}

	if deps.DownloadVersion == nil || deps.DownloadArchive == nil || deps.ExtractTarGz == nil || deps.RunInit == nil {
		return fmt.Errorf("installctl: missing required deps")
	}

	versionURL := strings.TrimRight(baseURL, "/") + "/version"
	version, err := deps.DownloadVersion(versionURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(version) != "" {
		fmt.Fprintf(out, "Detected remote ShellCrash version: %s\n", strings.TrimSpace(version))
	}

	archiveURL := strings.TrimRight(baseURL, "/") + "/ShellCrash.tar.gz"
	archivePath := filepath.Join(tmpDir, "ShellCrash.tar.gz")
	if err := deps.DownloadArchive(archiveURL, archivePath); err != nil {
		return err
	}
	defer func() { _ = os.Remove(archivePath) }()

	if err := deps.ExtractTarGz(archivePath, crashDir); err != nil {
		return err
	}

	if err := deps.RunInit(initctl.Options{
		CrashDir: crashDir,
		TmpDir:   tmpDir,
		FSRoot:   fsRoot,
	}); err != nil {
		return err
	}

	fmt.Fprintln(out, "ShellCrash Go installer completed successfully.")
	fmt.Fprintf(out, "CrashDir: %s\n", crashDir)
	return nil
}

