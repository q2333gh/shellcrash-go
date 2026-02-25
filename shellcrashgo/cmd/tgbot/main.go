package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"shellcrash/internal/tgbot"
)

func main() {
	defaultCrashDir := os.Getenv("CRASHDIR")
	rootFlags := flag.NewFlagSet("shellcrash-tgbot", flag.ExitOnError)
	crashDir := rootFlags.String("crashdir", defaultCrashDir, "ShellCrash root directory")
	_ = rootFlags.Parse(os.Args[1:])
	args := rootFlags.Args()

	var err error
	switch firstArg(args) {
	case "bind-chatid":
		err = runBindChatID(*crashDir, args[1:])
	case "set-push":
		err = runSetPush(*crashDir, args[1:])
	default:
		err = tgbot.Run(tgbot.Options{CrashDir: *crashDir}, tgbot.Deps{})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.TrimSpace(args[0])
}

func runBindChatID(_ string, args []string) error {
	fs := flag.NewFlagSet("bind-chatid", flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})
	url := fs.String("url", "", "telegram getUpdates URL")
	key := fs.String("key", "", "chat bind key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	chatID, err := tgbot.LookupChatID(*url, *key, tgbot.BindDeps{})
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, chatID)
	return nil
}

func runSetPush(crashDir string, args []string) error {
	fs := flag.NewFlagSet("set-push", flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})
	token := fs.String("token", "", "telegram bot token")
	chatID := fs.String("chat-id", "", "telegram chat ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return tgbot.SavePushConfig(crashDir, *token, *chatID)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
