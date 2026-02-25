package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/taskctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	flag.Parse()

	r := taskctl.Runner{CrashDir: *crashDir}
	if err := r.Run(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
