package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/installpathctl"
)

func main() {
	sysType := flag.String("systype", os.Getenv("systype"), "detected system type")
	envFile := flag.String("env-file", "", "write selected dir/crashdir shell vars to file")
	flag.Parse()

	action := "select"
	if len(flag.Args()) > 0 {
		action = flag.Args()[0]
	}

	opts := installpathctl.Options{
		SysType: *sysType,
		In:      os.Stdin,
		Out:     os.Stdout,
	}
	var (
		res installpathctl.Result
		err error
	)
	switch action {
	case "select":
		res, err = installpathctl.RunSelect(opts)
	case "usb":
		res, err = installpathctl.RunUSB(opts)
	case "xiaomi":
		res, err = installpathctl.RunXiaomi(opts)
	case "asus-usb":
		res, err = installpathctl.RunAsusUSB(opts)
	case "asus":
		res, err = installpathctl.RunAsus(opts)
	case "custom":
		res, err = installpathctl.RunCustom(opts)
	default:
		usageAndExit()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *envFile != "" {
		content := "dir=" + shellQuote(res.Dir) + "\nCRASHDIR=" + shellQuote(res.CrashDir) + "\n"
		if writeErr := os.WriteFile(*envFile, []byte(content), 0o600); writeErr != nil {
			fmt.Fprintln(os.Stderr, writeErr)
			os.Exit(1)
		}
	}
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage: installpathctl [--systype TYPE] [--env-file PATH] <select|usb|xiaomi|asus-usb|asus|custom>")
	os.Exit(2)
}

func shellQuote(v string) string {
	return "'" + escapeSingleQuotes(v) + "'"
}

func escapeSingleQuotes(v string) string {
	out := make([]byte, 0, len(v)+8)
	for i := 0; i < len(v); i++ {
		if v[i] == '\'' {
			out = append(out, '\'', '\\', '\'', '\'', '\'')
			continue
		}
		out = append(out, v[i])
	}
	return string(out)
}
