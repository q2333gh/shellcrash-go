package tgbot

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type BindDeps struct {
	HTTPDo func(*http.Request) (*http.Response, error)
}

var chatIDRegex = regexp.MustCompile(`"id"\s*:\s*(-?[0-9]+)\s*,\s*"is_bot"`)

func LookupChatID(updatesURL, key string, deps BindDeps) (string, error) {
	updatesURL = strings.TrimSpace(updatesURL)
	key = strings.TrimSpace(key)
	if updatesURL == "" {
		return "", fmt.Errorf("updates url is empty")
	}
	if key == "" {
		return "", fmt.Errorf("public key is empty")
	}
	if deps.HTTPDo == nil {
		client := &http.Client{Timeout: 15 * time.Second}
		deps.HTTPDo = client.Do
	}
	req, err := http.NewRequest(http.MethodGet, updatesURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := deps.HTTPDo(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("telegram updates status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	raw := string(body)
	if !strings.Contains(raw, key) {
		return "", fmt.Errorf("chat id not found for key")
	}
	m := chatIDRegex.FindStringSubmatch(raw)
	if len(m) < 2 {
		return "", fmt.Errorf("chat id not found in updates")
	}
	return m[1], nil
}

func SavePushConfig(crashDir, token, chatID string) error {
	crashDir = strings.TrimSpace(crashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)
	if token == "" {
		return fmt.Errorf("token is empty")
	}
	if chatID == "" {
		return fmt.Errorf("chat id is empty")
	}
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	if _, err := os.Stat(cfgPath); err != nil {
		return err
	}
	return setConfigValues(cfgPath, map[string]string{
		"push_TG": token,
		"chat_ID": chatID,
	})
}
