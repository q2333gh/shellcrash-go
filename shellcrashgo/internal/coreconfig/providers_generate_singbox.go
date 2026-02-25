package coreconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunProvidersGenerateSingbox ports singbox providers template generation from legacy shell.
func RunProvidersGenerateSingbox(opts Options, args []string) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	tmpDir := strings.TrimSpace(opts.TmpDir)
	if tmpDir == "" {
		tmpDir = "/tmp/ShellCrash"
	}
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}

	entries, tags, err := providersFromArgsOrConfig(crashDir, args)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("providers list is empty")
	}

	templatePath, err := resolveProviderTemplatePath(crashDir, "singbox", cfg)
	if err != nil {
		return err
	}
	tplData, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	tplObj, err := parseSingboxTemplateWithTags(string(tplData), tags)
	if err != nil {
		return err
	}

	basePath := filepath.Join(tmpDir, "config.json")
	merged := map[string]any{}
	if baseBytes, err := os.ReadFile(basePath); err == nil {
		if err := json.Unmarshal(baseBytes, &merged); err != nil {
			return fmt.Errorf("parse base config.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	skipCert := !strings.EqualFold(stripQuotes(cfg["skip_cert"]), "OFF")
	providers := buildSingboxProviders(entries, skipCert)
	outboundsAdd := buildSingboxProviderOutbounds(entries)
	mergeSingboxConfig(merged, tplObj, providers, outboundsAdd)

	output, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "jsons"), 0o755); err != nil {
		return err
	}
	target := filepath.Join(crashDir, "jsons", "config.json")
	if err := os.WriteFile(target, append(output, '\n'), 0o644); err != nil {
		return err
	}
	return nil
}

func parseSingboxTemplateWithTags(template string, tags []string) (map[string]any, error) {
	t := strings.TrimSpace(template)
	if strings.HasPrefix(t, "//") {
		if idx := strings.IndexByte(t, '\n'); idx >= 0 {
			t = strings.TrimSpace(t[idx+1:])
		}
	}
	quoted := make([]string, 0, len(tags))
	for _, tag := range tags {
		b, _ := json.Marshal(tag)
		quoted = append(quoted, string(b))
	}
	tagList := strings.Join(quoted, ", ")
	t = strings.ReplaceAll(t, "{providers_tags}", tagList)
	t = strings.ReplaceAll(t, `"providers_tags"`, tagList)

	out := map[string]any{}
	if err := json.Unmarshal([]byte(t), &out); err != nil {
		return nil, fmt.Errorf("parse singbox provider template: %w", err)
	}
	return out, nil
}

func buildSingboxProviders(entries []providerGenerateEntry, skipCert bool) []any {
	providers := make([]any, 0, len(entries))
	for _, e := range entries {
		hc := map[string]any{
			"enabled":  true,
			"url":      "https://www.gstatic.com/generate_204",
			"interval": fmt.Sprintf("%dm", e.Interval),
			"timeout":  "3s",
		}
		ot := map[string]any{
			"enabled":  true,
			"insecure": skipCert,
		}

		provider := map[string]any{
			"tag":          e.Tag,
			"health_check": hc,
			"override_tls": ot,
		}
		if strings.HasPrefix(e.Link, "./") {
			provider["type"] = "local"
			provider["path"] = e.Link
		} else {
			provider["type"] = "remote"
			provider["url"] = e.Link
			provider["path"] = "./providers/" + e.Tag + ".yaml"
			provider["user_agent"] = e.UA
			provider["update_interval"] = fmt.Sprintf("%dh", e.Interval2)
			if e.Exclude != "" {
				provider["exclude"] = e.Exclude
			}
			if e.Include != "" {
				provider["include"] = e.Include
			}
		}
		providers = append(providers, provider)
	}
	return providers
}

func buildSingboxProviderOutbounds(entries []providerGenerateEntry) []any {
	outbounds := make([]any, 0, len(entries))
	for _, e := range entries {
		outbounds = append(outbounds, map[string]any{
			"tag":       e.Tag,
			"type":      "urltest",
			"tolerance": float64(100),
			"providers": []any{e.Tag},
			"include":   ".*",
		})
	}
	return outbounds
}

func mergeSingboxConfig(base map[string]any, template map[string]any, providers []any, outboundsAdd []any) {
	baseOut := asAnySlice(base["outbounds"])
	tplOut := asAnySlice(template["outbounds"])
	mergedOut := make([]any, 0, len(baseOut)+len(outboundsAdd)+len(tplOut))
	mergedOut = append(mergedOut, baseOut...)
	mergedOut = append(mergedOut, outboundsAdd...)
	mergedOut = append(mergedOut, tplOut...)
	if len(mergedOut) > 0 {
		base["outbounds"] = mergedOut
	}

	baseProviders := asAnySlice(base["providers"])
	if len(baseProviders) > 0 || len(providers) > 0 {
		mergedProviders := make([]any, 0, len(baseProviders)+len(providers))
		mergedProviders = append(mergedProviders, baseProviders...)
		mergedProviders = append(mergedProviders, providers...)
		base["providers"] = mergedProviders
	}

	for k, v := range template {
		if k == "outbounds" {
			continue
		}
		base[k] = v
	}
}

func asAnySlice(v any) []any {
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}
