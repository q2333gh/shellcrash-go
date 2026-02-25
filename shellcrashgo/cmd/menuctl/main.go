package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/menuctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	binDir := flag.String("bindir", os.Getenv("BINDIR"), "ShellCrash binary directory")
	alias := flag.String("alias", os.Getenv("my_alias"), "ShellCrash alias to clean from profile")
	flag.Parse()

	args := flag.Args()
	mode := "menu"
	if len(args) > 0 {
		mode = args[0]
	}

	var err error
	switch mode {
	case "menu", "-l":
		err = menuctl.Run(menuctl.Options{
			CrashDir: *crashDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		})
	case "-s":
		if len(args) < 2 {
			usageAndExit()
		}
		err = menuctl.RunStartAction(*crashDir, args[1], args[2:]...)
	case "-i":
		err = menuctl.RunStartAction(*crashDir, "init")
	case "-d":
		debugArgs := []string{}
		if len(args) > 1 {
			debugArgs = args[1:]
		}
		err = menuctl.RunStartAction(*crashDir, "debug", debugArgs...)
	case "-u":
		err = menuctl.RunUninstallMenu(*crashDir, *binDir, *alias, os.Stdin, os.Stdout)
	default:
		usageAndExit()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  menuctl [menu|-l]")
	fmt.Fprintln(os.Stderr, "  menuctl -s <startctl-action> [args...]")
	fmt.Fprintln(os.Stderr, "  menuctl -i")
	fmt.Fprintln(os.Stderr, "  menuctl -d [level] [flash]")
	fmt.Fprintln(os.Stderr, "  menuctl -u")
	os.Exit(2)
}
