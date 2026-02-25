package main

import (
	"fmt"
	"os"

	"shellcrash/internal/minimalflow"
)

func main() {
	opts := minimalflow.Options{
		CrashDir: os.Getenv("CRASHDIR"),
		TmpDir:   os.Getenv("TMPDIR"),
	}
	if err := minimalflow.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("ok: generated %s\n", opts.CrashDir+"/yamls/config.yaml")
}
