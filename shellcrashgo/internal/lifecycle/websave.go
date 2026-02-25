package lifecycle

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var webSaveHTTPDo = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

type proxiesResponse struct {
	Proxies map[string]proxyItem `json:"proxies"`
}

type proxyItem struct {
	Type string   `json:"type"`
	All  []string `json:"all"`
	Now  string   `json:"now"`
}

// SaveWebSelections snapshots changed selector choices into configs/web_save.
func SaveWebSelections(crashDir, tmpDir, dbPort, secret string) error {
	if strings.TrimSpace(crashDir) == "" || strings.TrimSpace(dbPort) == "" {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+dbPort+"/proxies", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(secret) != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := webSaveHTTPDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	var decoded proxiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil
	}

	lines := buildWebSaveLines(decoded.Proxies)
	target := filepath.Join(crashDir, "configs", "web_save")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	old, _ := os.ReadFile(target)
	if string(old) == content {
		return nil
	}
	return os.WriteFile(target, []byte(content), 0o644)
}

func buildWebSaveLines(proxies map[string]proxyItem) []string {
	if len(proxies) == 0 {
		return nil
	}
	keys := make([]string, 0, len(proxies))
	for k := range proxies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, name := range keys {
		item := proxies[name]
		if !strings.EqualFold(strings.TrimSpace(item.Type), "Selector") {
			continue
		}
		if len(item.All) == 0 {
			continue
		}
		now := strings.TrimSpace(item.Now)
		def := strings.TrimSpace(item.All[0])
		if now == "" || def == now {
			continue
		}
		out = append(out, strings.TrimSpace(name)+","+now)
	}
	return out
}
