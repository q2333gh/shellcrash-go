package tgbot

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

type BindMenuDeps struct {
	LookupChatID   func(updatesURL, key string, deps BindDeps) (string, error)
	SavePushConfig func(crashDir, token, chatID string) error
	ReadBootID     func() (string, error)
	Sleep          func(time.Duration)
}

var bindChatIDPattern = regexp.MustCompile(`^[0-9]{8,}$`)

func RunBindMenu(opts Options, mode string, in io.Reader, out io.Writer, deps BindMenuDeps) error {
	opts.CrashDir = strings.TrimSpace(opts.CrashDir)
	if opts.CrashDir == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	if deps.LookupChatID == nil {
		deps.LookupChatID = LookupChatID
	}
	if deps.SavePushConfig == nil {
		deps.SavePushConfig = SavePushConfig
	}
	if deps.ReadBootID == nil {
		deps.ReadBootID = readBootID
	}
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}

	reader := bufio.NewReader(in)
	selectedMode := strings.TrimSpace(strings.ToLower(mode))
	if selectedMode == "" {
		fmt.Fprintln(out, "Telegram机器人绑定")
		fmt.Fprintln(out, "1) 私有机器人")
		fmt.Fprintln(out, "2) 公共机器人")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch strings.TrimSpace(choice) {
		case "", "0":
			return nil
		case "1":
			selectedMode = "private"
		case "2":
			selectedMode = "public"
		default:
			return fmt.Errorf("invalid selection")
		}
	}

	var token, updatesURL string
	switch selectedMode {
	case "private":
		fmt.Fprintln(out, "请先通过 https://t.me/BotFather 申请TG机器人并获取API TOKEN")
		fmt.Fprint(out, "请输入你获取到的API TOKEN(0返回)> ")
		text, err := readLine(reader)
		if err != nil {
			return err
		}
		token = strings.TrimSpace(text)
		if token == "" || token == "0" {
			return nil
		}
		updatesURL = "https://api.telegram.org/bot" + token + "/getUpdates"
		fmt.Fprintln(out, "请向你申请的机器人发送下面显示的秘钥")
	case "public":
		token = "publictoken"
		updatesURL = "https://tgbot.jwsc.eu.org/publictoken/getUpdates"
		fmt.Fprintln(out, "请向机器人 https://t.me/ShellCrashtg_bot 发送下面显示的秘钥")
	default:
		return fmt.Errorf("unsupported mode %q", selectedMode)
	}

	key, err := deps.ReadBootID()
	if err != nil || strings.TrimSpace(key) == "" {
		key = randomKey()
	}
	key = strings.TrimSpace(key)
	fmt.Fprintf(out, "发送此秘钥: %s\n", key)
	fmt.Fprint(out, "我已经发送完成(1/0)> ")
	confirm, err := readLine(reader)
	if err != nil {
		return err
	}
	if strings.TrimSpace(confirm) != "1" {
		return nil
	}

	chatID := ""
	for i := 1; i <= 3 && chatID == ""; i++ {
		chatID, _ = deps.LookupChatID(updatesURL, key, BindDeps{})
		chatID = strings.TrimSpace(chatID)
		if chatID != "" {
			break
		}
		if i < 3 {
			fmt.Fprintf(out, "第 %d 次尝试获取对话ID失败，正在重试...\n", i)
			deps.Sleep(time.Second)
		}
	}

	if chatID == "" && selectedMode != "public" {
		fmt.Fprintf(out, "无法自动获取对话ID，可手动访问 %s 查看\n", updatesURL)
		fmt.Fprint(out, "请手动输入ChatID(0返回)> ")
		text, err := readLine(reader)
		if err != nil {
			return err
		}
		chatID = strings.TrimSpace(text)
		if chatID == "" || chatID == "0" {
			return nil
		}
	}

	if !bindChatIDPattern.MatchString(chatID) {
		return fmt.Errorf("invalid chat id")
	}
	if err := deps.SavePushConfig(opts.CrashDir, token, chatID); err != nil {
		return err
	}
	fmt.Fprintln(out, "已完成Telegram日志推送设置")
	return nil
}

func readBootID() (string, error) {
	data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(data))
	idx := strings.LastIndex(text, "-")
	if idx >= 0 && idx+1 < len(text) {
		return text[idx+1:], nil
	}
	return text, nil
}

func randomKey() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(buf)
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
