package lifecycle

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreWebSelections(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "Group A,Node 1\nGroup/B,Node \"2\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "web_save"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	oldDo := webRestoreHTTPDo
	defer func() { webRestoreHTTPDo = oldDo }()

	var calls []string
	webRestoreHTTPDo = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.Header.Get("Authorization") != "Bearer x-secret" {
			t.Fatalf("missing auth header: %q", req.Header.Get("Authorization"))
		}
		b, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		calls = append(calls, req.URL.EscapedPath()+"|"+strings.TrimSpace(string(b)))
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	RestoreWebSelections(td, "10090", "x-secret")

	if len(calls) != 2 {
		t.Fatalf("expected 2 restore calls, got %d (%v)", len(calls), calls)
	}
	if !strings.Contains(calls[0], "/proxies/Group%20A|{\"name\":\"Node 1\"}") {
		t.Fatalf("unexpected first call: %s", calls[0])
	}
	if !strings.Contains(calls[1], "/proxies/Group%2FB|{\"name\":\"Node \\\"2\\\"\"}") {
		t.Fatalf("unexpected second call: %s", calls[1])
	}
}
