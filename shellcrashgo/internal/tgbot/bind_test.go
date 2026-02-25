package tgbot

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookupChatIDFromUpdates(t *testing.T) {
	body := `{"ok":true,"result":[{"message":{"text":"hello test-key","chat":{"id":123456789,"is_bot":false}}}]}`
	got, err := LookupChatID("http://example.test/getUpdates", "test-key", BindDeps{
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("LookupChatID failed: %v", err)
	}
	if got != "123456789" {
		t.Fatalf("unexpected chat id: %q", got)
	}
}

func TestLookupChatIDNotFound(t *testing.T) {
	body := `{"ok":true,"result":[{"message":{"text":"other","chat":{"id":987654321,"is_bot":false}}}]}`
	_, err := LookupChatID("http://example.test/getUpdates", "missing-key", BindDeps{
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		},
	})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestSavePushConfig(t *testing.T) {
	crashDir := t.TempDir()
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SavePushConfig(crashDir, "publictoken", "11223344"); err != nil {
		t.Fatalf("SavePushConfig failed: %v", err)
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "push_TG=publictoken") {
		t.Fatalf("missing push_TG in cfg: %s", text)
	}
	if !strings.Contains(text, "chat_ID=11223344") {
		t.Fatalf("missing chat_ID in cfg: %s", text)
	}
}
