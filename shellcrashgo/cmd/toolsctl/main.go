package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/toolsctl"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	flag.Parse()

	opts := toolsctl.Options{CrashDir: *crashDir}
	args := flag.Args()
	if len(args) == 0 || args[0] == "tools" {
		if err := toolsctl.RunToolsMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "ssh-tools" {
		if err := toolsctl.RunSSHToolsMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "mi-auto-ssh" {
		if err := toolsctl.RunMiAutoSSH(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "mi-ota-update" {
		action, err := toolsctl.ToggleMiOtaUpdate(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "已%s小米路由器的自动更新，如未生效，请在官方APP中同步设置！\n", action)
		return
	}
	if args[0] == "log-pusher" {
		if err := toolsctl.RunLogPusherMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "mi-tunfix" {
		action, err := toolsctl.ToggleMiTunfix(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if action == "enabled" {
			fmt.Fprintln(os.Stdout, "Tun模块补丁已启用，请重启服务。")
		} else {
			fmt.Fprintln(os.Stdout, "Tun模块补丁已禁用，请重启设备。")
		}
		return
	}
	if args[0] == "testcommand" {
		if err := toolsctl.RunTestCommandMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "debug" {
		if err := toolsctl.RunDebugMenu(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "userguide" {
		if err := toolsctl.RunUserguide(opts, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	usageAndExit()
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  toolsctl [tools|ssh-tools|mi-auto-ssh|mi-ota-update|log-pusher|mi-tunfix|testcommand|debug|userguide]")
	os.Exit(2)
}
