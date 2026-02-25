package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/firewall"
)

func main() {
	crashDir := flag.String("crashdir", "", "ShellCrash root directory")
	flag.Parse()

	vars, err := firewall.GetHostVars(*crashDir, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("host_ipv4=%s\n", shellQuote(vars.HostIPv4))
	fmt.Printf("host_ipv6=%s\n", shellQuote(vars.HostIPv6))
	fmt.Printf("local_ipv4=%s\n", shellQuote(vars.LocalIPv4))
	fmt.Printf("reserve_ipv4=%s\n", shellQuote(vars.ReserveIPv4))
	fmt.Printf("reserve_ipv6=%s\n", shellQuote(vars.ReserveIPv6))
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	out := "'"
	for _, r := range s {
		if r == '\'' {
			out += "'\\''"
			continue
		}
		out += string(r)
	}
	return out + "'"
}
