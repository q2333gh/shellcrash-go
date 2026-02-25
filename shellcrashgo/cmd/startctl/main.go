package main

import (
	"fmt"
	"os"

	"shellcrash/internal/startctl"
)

func main() {
	action := ""
	if len(os.Args) > 1 {
		action = os.Args[1]
	}
	level := ""
	flash := false
	extraArgs := []string{}
	if action == "debug" {
		if len(os.Args) > 2 {
			level = os.Args[2]
		}
		if len(os.Args) > 3 && os.Args[3] == "flash" {
			flash = true
		}
	} else if len(os.Args) > 2 {
		extraArgs = os.Args[2:]
	}

	cfg, err := startctl.LoadConfig(os.Getenv("CRASHDIR"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	if err := ctl.RunWithArgs(action, level, flash, extraArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
