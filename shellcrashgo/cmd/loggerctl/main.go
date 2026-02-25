package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/logger"
)

func main() {
	message := flag.String("message", "", "Log message")
	color := flag.String("color", "", "Display color code")
	push := flag.Bool("push", true, "Enable remote push notifications")
	overwrite := flag.Bool("overwrite", false, "Overwrite previous log entry with same message")
	tmpDir := flag.String("tmpdir", "/tmp/ShellCrash", "Temporary directory for logs")

	deviceName := flag.String("device-name", "", "Device name for push notifications")
	pushTG := flag.String("push-tg", "", "Telegram bot token")
	chatID := flag.String("chat-id", "", "Telegram chat ID")
	pushBark := flag.String("push-bark", "", "Bark push URL")
	pushDeer := flag.String("push-deer", "", "PushDeer key")
	pushPo := flag.String("push-po", "", "Pushover token")
	pushPoKey := flag.String("push-po-key", "", "Pushover user key")
	pushPP := flag.String("push-pp", "", "PushPlus token")
	pushGotify := flag.String("push-gotify", "", "Gotify URL")
	pushSynoChat := flag.String("push-synochat", "", "Synology Chat enabled")
	pushChatURL := flag.String("push-chat-url", "", "Synology Chat URL")
	pushChatToken := flag.String("push-chat-token", "", "Synology Chat token")
	pushChatUserID := flag.String("push-chat-userid", "", "Synology Chat user ID")

	flag.Parse()

	if *message == "" {
		fmt.Fprintln(os.Stderr, "Error: message is required")
		flag.Usage()
		os.Exit(1)
	}

	config := &logger.Config{
		DeviceName:     *deviceName,
		PushTG:         *pushTG,
		ChatID:         *chatID,
		PushBark:       *pushBark,
		PushDeer:       *pushDeer,
		PushPo:         *pushPo,
		PushPoKey:      *pushPoKey,
		PushPP:         *pushPP,
		PushGotify:     *pushGotify,
		PushSynoChat:   *pushSynoChat,
		PushChatURL:    *pushChatURL,
		PushChatToken:  *pushChatToken,
		PushChatUserID: *pushChatUserID,
	}

	l := logger.New(*tmpDir, config)
	if err := l.Log(*message, *color, *push, *overwrite); err != nil {
		fmt.Fprintf(os.Stderr, "Error logging: %v\n", err)
		os.Exit(1)
	}
}
