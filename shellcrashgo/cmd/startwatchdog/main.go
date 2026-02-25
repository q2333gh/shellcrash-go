package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/watchdog"
)

func main() {
	defaultCrashDir := os.Getenv("CRASHDIR")
	crashDir := flag.String("crashdir", defaultCrashDir, "ShellCrash root directory")
	flag.Parse()

	target := ""
	if flag.NArg() > 0 {
		target = flag.Arg(0)
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "usage: startwatchdog [--crashdir DIR] <target>")
		os.Exit(2)
	}

	if err := watchdog.Run(watchdog.Options{
		CrashDir: *crashDir,
		Target:   target,
	}, watchdog.Deps{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
