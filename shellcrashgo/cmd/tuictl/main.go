package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"shellcrash/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: tuictl <command> [args...]")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  separator <type>           - Print separator line (= or -)")
		fmt.Fprintln(os.Stderr, "  content <text>             - Print content line")
		fmt.Fprintln(os.Stderr, "  sub-content <text>         - Print sub-content line")
		fmt.Fprintln(os.Stderr, "  line-break                 - Print line break")
		fmt.Fprintln(os.Stderr, "  msg-alert <sleep> <msg...> - Print alert message")
		fmt.Fprintln(os.Stderr, "  comp-box <msg...>          - Print complete box")
		fmt.Fprintln(os.Stderr, "  top-box <msg...>           - Print top box")
		fmt.Fprintln(os.Stderr, "  btm-box <msg...>           - Print bottom box")
		fmt.Fprintln(os.Stderr, "  list-box <suffix> <items...> - Print numbered list")
		fmt.Fprintln(os.Stderr, "  common-success [msg]       - Print success message")
		fmt.Fprintln(os.Stderr, "  common-failed [msg]        - Print failure message")
		fmt.Fprintln(os.Stderr, "  common-back [msg]          - Print back option")
		fmt.Fprintln(os.Stderr, "  error-num [msg]            - Print number error")
		fmt.Fprintln(os.Stderr, "  error-letter [msg]         - Print letter error")
		fmt.Fprintln(os.Stderr, "  error-input [msg]          - Print input error")
		fmt.Fprintln(os.Stderr, "  cancel-back [msg]          - Print cancel message")
		os.Exit(1)
	}

	tw := tui.NewWriter(os.Stdout)
	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "separator":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Error: separator requires type (= or -)")
			os.Exit(1)
		}
		tw.SeparatorLine(args[0])

	case "content":
		text := ""
		if len(args) > 0 {
			text = strings.Join(args, " ")
		}
		tw.ContentLine(text)

	case "sub-content":
		text := ""
		if len(args) > 0 {
			text = strings.Join(args, " ")
		}
		tw.SubContentLine(text)

	case "line-break":
		tw.LineBreak()

	case "msg-alert":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Error: msg-alert requires sleep time")
			os.Exit(1)
		}
		sleepTime, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid sleep time: %v\n", err)
			os.Exit(1)
		}
		messages := args[1:]
		tw.MsgAlert(sleepTime, messages...)

	case "comp-box":
		tw.CompBox(args...)

	case "top-box":
		tw.TopBox(args...)

	case "btm-box":
		tw.BtmBox(args...)

	case "list-box":
		suffix := ""
		if len(args) > 0 {
			suffix = args[0]
			args = args[1:]
		}
		tw.ListBox(args, suffix)

	case "common-success":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.CommonSuccess(msg)

	case "common-failed":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.CommonFailed(msg)

	case "common-back":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.CommonBack(msg)

	case "error-num":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.ErrorNum(msg)

	case "error-letter":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.ErrorLetter(msg)

	case "error-input":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.ErrorInput(msg)

	case "cancel-back":
		msg := ""
		if len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		tw.CancelBack(msg)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}
