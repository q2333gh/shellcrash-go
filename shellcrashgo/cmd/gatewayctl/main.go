package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/gatewayctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	flag.Parse()

	opts := gatewayctl.Options{CrashDir: *crashDir}
	args := flag.Args()
	if len(args) == 0 || args[0] == "menu" {
		if err := gatewayctl.RunGatewayMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	switch args[0] {
	case "fw-filter":
		if err := gatewayctl.RunTrafficFilterMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "fw-wan":
		if err := gatewayctl.RunFirewallMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "lan-filter":
		if err := gatewayctl.RunLANDeviceFilterMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "tg-menu":
		if err := gatewayctl.RunTGMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "vmess":
		if err := gatewayctl.RunVmessMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "common-ports":
		if err := gatewayctl.RunCommonPortsMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "cust-host-ipv4":
		if err := gatewayctl.RunCustomHostIPv4Menu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "reserve-ipv4":
		if err := gatewayctl.RunReserveIPv4Menu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "sss":
		if err := gatewayctl.RunShadowsocksMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "tailscale":
		if err := gatewayctl.RunTailscaleMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "wireguard":
		if err := gatewayctl.RunWireGuardMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "tg-service":
		mode := "toggle"
		if len(args) > 1 {
			mode = args[1]
		}
		next, err := gatewayctl.SetTGService(opts, mode)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(next)
		return
	case "tg-menupush":
		mode := "toggle"
		if len(args) > 1 {
			mode = args[1]
		}
		next, err := gatewayctl.SetTGMenuPush(opts, mode)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(next)
		return
	case "tg-config":
		fs := flag.NewFlagSet("tg-config", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		token := fs.String("token", "", "telegram bot token")
		chatID := fs.String("chat-id", "", "telegram chat id")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		if err := gatewayctl.ConfigureTGBot(opts, *token, *chatID, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok")
		return
	}

	usageAndExit()
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  gatewayctl [menu]")
	fmt.Fprintln(os.Stderr, "  gatewayctl fw-filter")
	fmt.Fprintln(os.Stderr, "  gatewayctl fw-wan")
	fmt.Fprintln(os.Stderr, "  gatewayctl lan-filter")
	fmt.Fprintln(os.Stderr, "  gatewayctl common-ports")
	fmt.Fprintln(os.Stderr, "  gatewayctl cust-host-ipv4")
	fmt.Fprintln(os.Stderr, "  gatewayctl reserve-ipv4")
	fmt.Fprintln(os.Stderr, "  gatewayctl vmess")
	fmt.Fprintln(os.Stderr, "  gatewayctl sss")
	fmt.Fprintln(os.Stderr, "  gatewayctl tailscale")
	fmt.Fprintln(os.Stderr, "  gatewayctl wireguard")
	fmt.Fprintln(os.Stderr, "  gatewayctl tg-menu")
	fmt.Fprintln(os.Stderr, "  gatewayctl tg-service [toggle|on|off]")
	fmt.Fprintln(os.Stderr, "  gatewayctl tg-menupush [toggle|on|off]")
	fmt.Fprintln(os.Stderr, "  gatewayctl tg-config --token <token> --chat-id <id>")
	os.Exit(2)
}
