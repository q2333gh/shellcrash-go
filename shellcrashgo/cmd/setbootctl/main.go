package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"shellcrash/internal/setbootctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	flag.Parse()

	opts := setbootctl.Options{CrashDir: *crashDir}
	args := flag.Args()
	if len(args) == 0 || args[0] == "menu" {
		if err := setbootctl.RunMenu(opts, setbootctl.Deps{}, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "autostart":
		if len(args) < 2 {
			usageAndExit()
		}
		switch args[1] {
		case "enable":
			exitIfErr(setbootctl.EnableAutostart(opts, setbootctl.Deps{}))
		case "disable":
			exitIfErr(setbootctl.DisableAutostart(opts, setbootctl.Deps{}))
		default:
			usageAndExit()
		}
	case "set-delay":
		if len(args) < 2 {
			usageAndExit()
		}
		sec, err := strconv.Atoi(args[1])
		if err != nil || sec < 0 || sec > 300 {
			exitIfErr(fmt.Errorf("invalid delay: %s", args[1]))
		}
		in := fmt.Sprintf("3\n%d\n0\n", sec)
		exitIfErr(setbootctl.RunMenu(opts, setbootctl.Deps{}, strings.NewReader(in), io.Discard))
	case "set-bindir":
		if len(args) < 2 {
			usageAndExit()
		}
		exitIfErr(setbootctl.SetBindDir(opts, args[1]))
	default:
		usageAndExit()
	}
}

func exitIfErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  setbootctl [menu]")
	fmt.Fprintln(os.Stderr, "  setbootctl autostart <enable|disable>")
	fmt.Fprintln(os.Stderr, "  setbootctl set-delay <0-300>")
	fmt.Fprintln(os.Stderr, "  setbootctl set-bindir <path>")
	os.Exit(2)
}
