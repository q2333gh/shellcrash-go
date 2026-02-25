package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/uninstallctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	binDir := flag.String("bindir", os.Getenv("BINDIR"), "ShellCrash binary directory")
	alias := flag.String("alias", os.Getenv("my_alias"), "ShellCrash alias to clean from profile")
	keepConfig := flag.Bool("keep-config", false, "keep configs/yamls/jsons under crashdir")
	fsRoot := flag.String("fsroot", "", "filesystem root override for testing")
	flag.Parse()

	opts := uninstallctl.Options{
		CrashDir:   *crashDir,
		BinDir:     *binDir,
		Alias:      *alias,
		KeepConfig: *keepConfig,
		FSRoot:     *fsRoot,
	}

	args := flag.Args()
	var err error
	if len(args) > 0 && args[0] == "menu" {
		err = uninstallctl.RunMenu(opts, uninstallctl.Deps{}, os.Stdin, os.Stdout)
	} else {
		err = uninstallctl.Run(opts, uninstallctl.Deps{})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
