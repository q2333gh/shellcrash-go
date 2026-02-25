package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/settingsctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	flag.Parse()

	opts := settingsctl.Options{CrashDir: *crashDir}
	args := flag.Args()
	mode := "menu"
	if len(args) > 0 {
		mode = args[0]
	}

	var err error
	switch mode {
	case "menu":
		err = settingsctl.RunMenu(opts, os.Stdin, os.Stdout)
	case "adv-ports":
		err = settingsctl.RunAdvancedPortMenu(opts, os.Stdin, os.Stdout)
	case "ipv6":
		err = settingsctl.RunIPv6Menu(opts, os.Stdin, os.Stdout)
	case "redir":
		err = settingsctl.RunMenu(opts, os.Stdin, os.Stdout)
	case "dns":
		err = settingsctl.RunDNSMenu(opts, os.Stdin, os.Stdout)
	case "dns-fakeip":
		err = settingsctl.RunFakeIPFilterMenu(opts, os.Stdin, os.Stdout)
	case "dns-adv":
		err = settingsctl.RunDNSAdvancedMenu(opts, os.Stdin, os.Stdout)
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
	fmt.Fprintln(os.Stderr, "  settingsctl [menu]")
	fmt.Fprintln(os.Stderr, "  settingsctl adv-ports")
	fmt.Fprintln(os.Stderr, "  settingsctl ipv6")
	fmt.Fprintln(os.Stderr, "  settingsctl dns")
	fmt.Fprintln(os.Stderr, "  settingsctl dns-fakeip")
	fmt.Fprintln(os.Stderr, "  settingsctl dns-adv")
	os.Exit(2)
}
