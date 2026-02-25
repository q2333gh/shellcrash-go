package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/upgradectl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	flag.Parse()

	opts := upgradectl.Options{CrashDir: *crashDir}
	args := flag.Args()
	if len(args) == 0 || args[0] == "menu" || args[0] == "upgrade" {
		if err := upgradectl.RunUpgradeMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "setserver" {
		if err := upgradectl.RunSetServerMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "setcrt" {
		if err := upgradectl.RunSetCertMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "setscripts" {
		if err := upgradectl.RunSetScriptsMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "setgeo" {
		if err := upgradectl.RunSetGeoMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "setdb" {
		if err := upgradectl.RunSetDBMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "setcore" {
		if err := upgradectl.RunSetCoreMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	usageAndExit()
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  upgradectl [menu|upgrade|setserver|setcrt|setscripts|setgeo|setdb|setcore]")
	os.Exit(2)
}
