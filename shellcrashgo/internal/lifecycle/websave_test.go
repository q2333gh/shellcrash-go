package lifecycle

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveWebSelectionsWritesChangedSelectors(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldDo := webSaveHTTPDo
	defer func() { webSaveHTTPDo = oldDo }()

	webSaveHTTPDo = func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/proxies" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer sec" {
			t.Fatalf("missing auth: %q", req.Header.Get("Authorization"))
		}
		body := `{"proxies":{"GLOBAL":{"type":"Selector","all":["DIRECT","Node A"],"now":"Node A"},"DIRECT":{"type":"Selector","all":["DIRECT"],"now":"DIRECT"},"AUTO":{"type":"URLTest","all":["Node A"],"now":"Node A"}}}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	}

	if err := SaveWebSelections(td, filepath.Join(td, "tmp"), "9090", "sec"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "web_save"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if got != "GLOBAL,Node A\n" {
		t.Fatalf("unexpected web_save content: %q", got)
	}
}

func TestSaveWebSelectionsClearsFileWhenNoChanges(t *testing.T) {
	td := t.TempDir()
	cfgDir := filepath.Join(td, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "web_save")
	if err := os.WriteFile(path, []byte("OLD,Node\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldDo := webSaveHTTPDo
	defer func() { webSaveHTTPDo = oldDo }()

	webSaveHTTPDo = func(req *http.Request) (*http.Response, error) {
		body := `{"proxies":{"GLOBAL":{"type":"Selector","all":["DIRECT"],"now":"DIRECT"}}}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	}

	if err := SaveWebSelections(td, filepath.Join(td, "tmp"), "9090", ""); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "" {
		t.Fatalf("expected cleared web_save, got %q", string(data))
	}
}
