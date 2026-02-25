package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/utils"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "webget":
		runWebGet()
	case "get-bin":
		runGetBin()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: webgetctl <command> [options]

Commands:
  webget <local-path> <remote-url> [options]
    Download a file from a URL to a local path
    Options:
      --proxy <url>          HTTP/HTTPS proxy URL
      --user-agent <ua>      Custom user agent
      --skip-cert            Skip certificate verification
      --no-redirect          Disable following redirects
      --timeout <seconds>    Timeout in seconds (default: 30)
      --rewrite-github       Rewrite GitHub URLs when proxy is active

  get-bin <local-path> <remote-path> [options]
    Download project-specific files using configured servers
    Options:
      --crashdir <dir>       ShellCrash directory (default: $CRASHDIR)
      --update-url <url>     Base update URL
      --url-id <id>          Server ID from configs/servers.list
      --release-type <type>  Release type (master, dev, update)
      --proxy <url>          HTTP/HTTPS proxy URL
      --user-agent <ua>      Custom user agent
      --timeout <seconds>    Timeout in seconds (default: 30)
`)
}

func runWebGet() {
	fs := flag.NewFlagSet("webget", flag.ExitOnError)
	proxy := fs.String("proxy", "", "HTTP/HTTPS proxy URL")
	userAgent := fs.String("user-agent", "", "Custom user agent")
	skipCert := fs.Bool("skip-cert", false, "Skip certificate verification")
	noRedirect := fs.Bool("no-redirect", false, "Disable following redirects")
	timeout := fs.Int("timeout", 30, "Timeout in seconds")
	rewriteGitHub := fs.Bool("rewrite-github", false, "Rewrite GitHub URLs when proxy is active")

	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: webgetctl webget <local-path> <remote-url> [options]\n")
		os.Exit(1)
	}

	localPath := args[0]
	remoteURL := args[1]

	opts := &utils.WebGetOptions{
		ProxyURL:      *proxy,
		UserAgent:     *userAgent,
		SkipCertCheck: *skipCert,
		NoRedirect:    *noRedirect,
		Timeout:       *timeout,
		RewriteGitHub: *rewriteGitHub,
	}

	err := utils.WebGet(localPath, remoteURL, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}
}

func runGetBin() {
	fs := flag.NewFlagSet("get-bin", flag.ExitOnError)
	crashDir := fs.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash directory")
	updateURL := fs.String("update-url", "", "Base update URL")
	urlID := fs.String("url-id", "", "Server ID from configs/servers.list")
	releaseType := fs.String("release-type", "", "Release type (master, dev, update)")
	proxy := fs.String("proxy", "", "HTTP/HTTPS proxy URL")
	userAgent := fs.String("user-agent", "", "Custom user agent")
	timeout := fs.Int("timeout", 30, "Timeout in seconds")

	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: webgetctl get-bin <local-path> <remote-path> [options]\n")
		os.Exit(1)
	}

	localPath := args[0]
	remotePath := args[1]

	if *crashDir == "" {
		fmt.Fprintf(os.Stderr, "Error: --crashdir is required or set $CRASHDIR\n")
		os.Exit(1)
	}

	webGetOpts := &utils.WebGetOptions{
		ProxyURL:  *proxy,
		UserAgent: *userAgent,
		Timeout:   *timeout,
	}

	opts := &utils.GetBinOptions{
		UpdateURL:   *updateURL,
		URLId:       *urlID,
		ReleaseType: *releaseType,
		WebGetOpts:  webGetOpts,
	}

	err := utils.GetBin(localPath, remotePath, *crashDir, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}
}
