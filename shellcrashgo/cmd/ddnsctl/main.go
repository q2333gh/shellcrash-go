package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"shellcrash/internal/ddnsctl"
)

func main() {
	configPath := flag.String("config", "/etc/config/ddns", "ddns config path")
	logDir := flag.String("log-dir", "/var/log/ddns", "ddns log directory")
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "shellcrash dir (unused, for wrapper compatibility)")
	flag.Parse()
	_ = crashDir

	args := flag.Args()
	if len(args) == 0 {
		if err := ddnsctl.RunMenu(ddnsctl.Options{
			ConfigPath: *configPath,
			LogDir:     *logDir,
		}, ddnsctl.Deps{}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	opts := ddnsctl.Options{ConfigPath: *configPath, LogDir: *logDir}
	switch args[0] {
	case "menu":
		err := ddnsctl.RunMenu(opts, ddnsctl.Deps{})
		exitIfErr(err)
	case "list":
		services, err := ddnsctl.ListServices(opts, ddnsctl.Deps{})
		exitIfErr(err)
		for _, s := range services {
			fmt.Printf("%s\t%s\t%s\t%s\n", s.Name, s.Domain, s.Enabled, s.LastIP)
		}
	case "add":
		if len(args) < 6 {
			usageAndExit()
		}
		checkInterval := 10
		forceInterval := 24
		if len(args) >= 7 {
			if n, err := strconv.Atoi(args[6]); err == nil {
				checkInterval = n
			}
		}
		if len(args) >= 8 {
			if n, err := strconv.Atoi(args[7]); err == nil {
				forceInterval = n
			}
		}
		err := ddnsctl.AddService(opts, ddnsctl.Deps{}, ddnsctl.AddParams{
			ServiceID:     args[1],
			ServiceName:   args[2],
			Domain:        args[3],
			Username:      args[4],
			Password:      args[5],
			CheckInterval: checkInterval,
			ForceInterval: forceInterval,
		})
		exitIfErr(err)
	case "update":
		if len(args) < 2 {
			usageAndExit()
		}
		exitIfErr(ddnsctl.UpdateService(opts, ddnsctl.Deps{}, args[1]))
	case "toggle":
		if len(args) < 2 {
			usageAndExit()
		}
		exitIfErr(ddnsctl.ToggleService(opts, ddnsctl.Deps{}, args[1]))
	case "remove":
		if len(args) < 2 {
			usageAndExit()
		}
		exitIfErr(ddnsctl.RemoveService(opts, ddnsctl.Deps{}, args[1]))
	case "log":
		if len(args) < 2 {
			usageAndExit()
		}
		exitIfErr(ddnsctl.PrintServiceLog(opts, args[1], os.Stdout))
	default:
		usageAndExit()
	}
}

func exitIfErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  ddnsctl [menu]")
	fmt.Fprintln(os.Stderr, "  ddnsctl list")
	fmt.Fprintln(os.Stderr, "  ddnsctl add <service_id> <service_name> <domain> <username> <password> [check_interval] [force_interval]")
	fmt.Fprintln(os.Stderr, "  ddnsctl update <service_id>")
	fmt.Fprintln(os.Stderr, "  ddnsctl toggle <service_id>")
	fmt.Fprintln(os.Stderr, "  ddnsctl remove <service_id>")
	fmt.Fprintln(os.Stderr, "  ddnsctl log <service_id>")
	os.Exit(2)
}
