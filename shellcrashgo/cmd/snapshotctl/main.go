package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/snapshotctl"
)

func main() {
	defaultCrashDir := os.Getenv("CRASHDIR")
	crashDir := flag.String("crashdir", defaultCrashDir, "ShellCrash root directory")
	fsRoot := flag.String("fsroot", "/", "filesystem root prefix")
	flag.Parse()

	action := "default"
	if flag.NArg() > 0 {
		action = flag.Arg(0)
	}

	if err := snapshotctl.Run(snapshotctl.Options{
		CrashDir: *crashDir,
		FSRoot:   *fsRoot,
		Action:   action,
	}, snapshotctl.Deps{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
