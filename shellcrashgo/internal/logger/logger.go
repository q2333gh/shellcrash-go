package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxLogLines = 200
	trimLines   = 20
)

type Config struct {
	DeviceName     string
	PushTG         string
	ChatID         string
	PushBark       string
	PushDeer       string
	PushPo         string
	PushPoKey      string
	PushPP         string
	PushGotify     string
	PushSynoChat   string
	PushChatURL    string
	PushChatToken  string
	PushChatUserID string
}

type Logger struct {
	tmpDir string
	config *Config
}

func New(tmpDir string, config *Config) *Logger {
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	if config == nil {
		config = &Config{}
	}
	return &Logger{
		tmpDir: tmpDir,
		config: config,
	}
}

func (l *Logger) Log(message string, color string, push bool, overwrite bool) error {
	if color != "" && color != "0" {
		fmt.Printf("\033[%sm%s\033[0m\n", color, message)
	}

	logFile := filepath.Join(l.tmpDir, "ShellCrash.log")
	timestamp := time.Now().Format("2006-01-02_15:04:05")
	logText := fmt.Sprintf("%s~%s", timestamp, message)

	if overwrite {
		if err := l.removeLineContaining(logFile, message); err != nil {
			return err
		}
	}

	if err := l.appendLog(logFile, logText); err != nil {
		return err
	}

	if err := l.trimLog(logFile); err != nil {
		return err
	}

	if push {
		go l.pushRemoteLogs(logText)
	}

	return nil
}

func (l *Logger) appendLog(logFile, logText string) error {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(logText + "\n")
	return err
}

func (l *Logger) removeLineContaining(logFile, pattern string) error {
	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, pattern) {
			filtered = append(filtered, line)
		}
	}

	return os.WriteFile(logFile, []byte(strings.Join(filtered, "\n")), 0644)
}

func (l *Logger) trimLog(logFile string) error {
	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLogLines {
		lines = lines[trimLines:]
		return os.WriteFile(logFile, []byte(strings.Join(lines, "\n")), 0644)
	}

	return nil
}

func (l *Logger) pushRemoteLogs(logText string) {
	if l.config.DeviceName != "" {
		logText = fmt.Sprintf("%s(%s)", logText, l.config.DeviceName)
	}

	if l.config.PushTG != "" {
		l.pushTelegram(logText)
	}
	if l.config.PushBark != "" {
		l.pushBark(logText)
	}
	if l.config.PushDeer != "" {
		l.pushDeer(logText)
	}
	if l.config.PushPo != "" {
		l.pushPushover(logText)
	}
	if l.config.PushPP != "" {
		l.pushPushPlus(logText)
	}
	if l.config.PushGotify != "" {
		l.pushGotify(logText)
	}
	if l.config.PushSynoChat != "" {
		l.pushSynoChat(logText)
	}
}

func (l *Logger) pushTelegram(logText string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", l.config.PushTG)
	if l.config.PushTG == "publictoken" {
		url = "https://tgbot.jwsc.eu.org/publictoken/sendMessage"
	}

	payload := map[string]string{
		"chat_id": l.config.ChatID,
		"text":    logText,
	}
	l.postJSON(url, payload)
}

func (l *Logger) pushBark(logText string) {
	payload := map[string]string{
		"body":  logText,
		"title": "ShellCrash_log",
		"level": "passive",
		"badge": "1",
	}
	l.postJSON(l.config.PushBark, payload)
}

func (l *Logger) pushDeer(logText string) {
	url := "https://api2.pushdeer.com/message/push"
	payload := map[string]string{
		"pushkey": l.config.PushDeer,
		"text":    logText,
	}
	l.postJSON(url, payload)
}

func (l *Logger) pushPushover(logText string) {
	url := "https://api.pushover.net/1/messages.json"
	payload := map[string]string{
		"token":   l.config.PushPo,
		"user":    l.config.PushPoKey,
		"title":   "ShellCrash_log",
		"message": logText,
	}
	l.postJSON(url, payload)
}

func (l *Logger) pushPushPlus(logText string) {
	url := "http://www.pushplus.plus/send"
	payload := map[string]string{
		"token":   l.config.PushPP,
		"title":   "ShellCrash_log",
		"content": logText,
	}
	l.postJSON(url, payload)
}

func (l *Logger) pushGotify(logText string) {
	payload := map[string]interface{}{
		"title":    "ShellCrash_log",
		"message":  logText,
		"priority": 5,
	}
	l.postJSON(l.config.PushGotify, payload)
}

func (l *Logger) pushSynoChat(logText string) {
	url := fmt.Sprintf("%s/webapi/entry.cgi?api=SYNO.Chat.External&method=chatbot&version=2&token=%s",
		l.config.PushChatURL, l.config.PushChatToken)

	payloadStr := fmt.Sprintf("payload={\"text\":\"%s\", \"user_ids\":[%s]}", logText, l.config.PushChatUserID)
	l.postFormData(url, payloadStr)
}

func (l *Logger) postJSON(url string, payload interface{}) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
}

func (l *Logger) postFormData(url, data string) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
}
