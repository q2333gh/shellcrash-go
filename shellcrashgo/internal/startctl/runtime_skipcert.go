package startctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func (c *Controller) enforceSingboxSkipCert(jsonDir string) error {
	want := !strings.EqualFold(c.Cfg.SkipCert, "OFF")
	entries, err := os.ReadDir(jsonDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(jsonDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var data any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		if !setInsecureRecursive(data, want) {
			continue
		}
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		out = append(out, '\n')
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func setInsecureRecursive(node any, value bool) bool {
	switch v := node.(type) {
	case map[string]any:
		changed := false
		for key, child := range v {
			if key == "insecure" {
				if cur, ok := child.(bool); !ok || cur != value {
					v[key] = value
					changed = true
				}
				continue
			}
			if setInsecureRecursive(child, value) {
				changed = true
			}
		}
		return changed
	case []any:
		changed := false
		for _, child := range v {
			if setInsecureRecursive(child, value) {
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}
