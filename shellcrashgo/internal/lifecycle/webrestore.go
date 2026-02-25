package lifecycle

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var webRestoreHTTPDo = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

// RestoreWebSelections replays saved dashboard proxy selections from configs/web_save.
func RestoreWebSelections(crashDir, dbPort, secret string) {
	if crashDir == "" || dbPort == "" {
		return
	}
	savePath := filepath.Join(crashDir, "configs", "web_save")
	data, err := os.ReadFile(savePath)
	if err != nil || len(data) == 0 {
		return
	}

	base := "http://127.0.0.1:" + dbPort + "/proxies/"
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		group, now, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		group = strings.TrimSpace(group)
		now = strings.TrimSpace(now)
		if group == "" || now == "" {
			continue
		}
		body := []byte(`{"name":"` + jsonEscape(now) + `"}`)
		req, err := http.NewRequest(http.MethodPut, base+url.PathEscape(group), bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			req.Header.Set("Authorization", "Bearer "+secret)
		}
		resp, err := webRestoreHTTPDo(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}
}

func jsonEscape(v string) string {
	r := strings.NewReplacer(
		`\\`, `\\\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	)
	return r.Replace(v)
}
