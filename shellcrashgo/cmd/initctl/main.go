package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/initctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	tmpDir := flag.String("tmpdir", os.Getenv("TMPDIR"), "runtime tmp directory")
	flag.Parse()

	opts := initctl.Options{CrashDir: *crashDir, TmpDir: *tmpDir}
	if err := initctl.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(initctl.PrintSummary(opts))
}
