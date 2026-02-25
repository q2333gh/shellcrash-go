package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"shellcrash/internal/utils"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "check-port":
		checkPortCmd()
	case "url-encode":
		urlEncodeCmd()
	case "url-decode":
		urlDecodeCmd()
	case "cmd-exists":
		cmdExistsCmd()
	case "compare-files":
		compareFilesCmd()
	case "core-unzip":
		coreUnzipCmd()
	case "core-find":
		coreFindCmd()
	case "core-check":
		coreCheckCmd()
	case "core-install":
		coreInstallCmd()
	case "core-target":
		coreTargetCmd()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: utilsctl <command> [args]

Commands:
  check-port <port> [used-ports...]  Check if a port is valid and available
  url-encode <string>                URL-encode a string
  url-decode <string>                URL-decode a string
  cmd-exists <command>               Check if a command exists in PATH
  compare-files <file1> <file2>      Compare two files for equality
  core-unzip <source> <target> <tmpdir> <bindir>  Extract core binary from archive
  core-find <tmpdir> <bindir> <crashdir>          Find and extract core archive
  core-check <archive> <tmpdir> <bindir> <crashdir> <crashcore>  Verify core binary
  core-install <archive> <tmpdir> <bindir> <ziptype> <version> <command>  Install verified core
  core-target <crashcore>            Get core target type and format
`)
}

func checkPortCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl check-port <port> [used-ports...]\n")
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid port number: %s\n", os.Args[2])
		os.Exit(1)
	}

	var usedPorts []int
	for _, arg := range os.Args[3:] {
		p, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid port number: %s\n", arg)
			os.Exit(1)
		}
		usedPorts = append(usedPorts, p)
	}

	if err := utils.CheckPort(port, usedPorts); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Println("Port is available")
}

func urlEncodeCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl url-encode <string>\n")
		os.Exit(1)
	}

	input := strings.Join(os.Args[2:], " ")
	fmt.Println(utils.URLEncode(input))
}

func urlDecodeCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl url-decode <string>\n")
		os.Exit(1)
	}

	input := strings.Join(os.Args[2:], " ")
	decoded, err := utils.URLDecode(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(decoded)
}

func cmdExistsCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl cmd-exists <command>\n")
		os.Exit(1)
	}

	cmd := os.Args[2]
	if utils.CommandExists(cmd) {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func compareFilesCmd() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl compare-files <file1> <file2>\n")
		os.Exit(1)
	}

	file1 := os.Args[2]
	file2 := os.Args[3]

	same, err := utils.CompareFiles(file1, file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error comparing files: %v\n", err)
		os.Exit(1)
	}

	if same {
		fmt.Println("Files are identical")
		os.Exit(0)
	} else {
		fmt.Println("Files are different")
		os.Exit(1)
	}
}

var _ = flag.Bool // silence unused import warning

func coreUnzipCmd() {
	if len(os.Args) < 6 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl core-unzip <source> <target> <tmpdir> <bindir>\n")
		os.Exit(1)
	}

	source := os.Args[2]
	target := os.Args[3]
	tmpDir := os.Args[4]
	binDir := os.Args[5]

	if err := utils.CoreUnzip(source, target, tmpDir, binDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unzip core: %v\n", err)
		os.Exit(1)
	}
}

func coreFindCmd() {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl core-find <tmpdir> <bindir> <crashdir>\n")
		os.Exit(1)
	}

	tmpDir := os.Args[2]
	binDir := os.Args[3]
	crashDir := os.Args[4]

	if err := utils.CoreFind(tmpDir, binDir, crashDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find core: %v\n", err)
		os.Exit(1)
	}
}

func coreCheckCmd() {
	if len(os.Args) < 7 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl core-check <archive> <tmpdir> <bindir> <crashdir> <crashcore>\n")
		os.Exit(1)
	}

	archive := os.Args[2]
	tmpDir := os.Args[3]
	binDir := os.Args[4]
	crashDir := os.Args[5]
	crashCore := os.Args[6]

	result, err := utils.CoreCheck(archive, tmpDir, binDir, crashDir, crashCore, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Core check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("version=%s\n", result.Version)
	fmt.Printf("command=%s\n", result.Command)
}

func coreInstallCmd() {
	if len(os.Args) < 8 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl core-install <archive> <tmpdir> <bindir> <ziptype> <version> <command>\n")
		os.Exit(1)
	}

	archive := os.Args[2]
	tmpDir := os.Args[3]
	binDir := os.Args[4]
	zipType := os.Args[5]
	version := os.Args[6]
	command := os.Args[7]

	result := &utils.CoreCheckResult{
		Version: version,
		Command: command,
		IsValid: true,
	}

	if err := utils.CoreInstall(archive, tmpDir, binDir, zipType, result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install core: %v\n", err)
		os.Exit(1)
	}
}

func coreTargetCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: utilsctl core-target <crashcore>\n")
		os.Exit(1)
	}

	crashCore := os.Args[2]
	target, format := utils.GetCoreTarget(crashCore)
	fmt.Printf("target=%s\n", target)
	fmt.Printf("format=%s\n", format)
}
