package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/legacylaunch"
)

func main() {
	command := flag.String("command", "", "command to run in background")
	name := flag.String("name", "", "pid file name without extension")
	tmpDir := flag.String("tmpdir", "/tmp/ShellCrash", "pid directory")
	flag.Parse()

	if err := legacylaunch.Run(legacylaunch.Options{Command: *command, Name: *name, TmpDir: *tmpDir}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
