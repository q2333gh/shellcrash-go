package coreconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type providerGenerateEntry struct {
	Tag       string
	Link      string
	Interval  int
	Interval2 int
	UA        string
	Exclude   string
	Include   string
	IsURI     bool
}

// RunProvidersGenerateClash ports providers template generation from legacy shell for clash cores.
func RunProvidersGenerateClash(opts Options, args []string) error {
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

	templatePath, err := resolveProviderTemplatePath(crashDir, "clash", cfg)
	if err != nil {
		return err
	}
	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	groupsSection, rulesSection, err := splitClashProviderTemplate(string(tpl))
	if err != nil {
		return err
	}

	tmpProvidersDir := filepath.Join(tmpDir, "providers")
	if err := os.MkdirAll(tmpProvidersDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(crashDir, "yamls"), 0o755); err != nil {
		return err
	}

	providerBlock := buildClashProvidersBlock(entries, cfg)
	groupsOut := strings.ReplaceAll(groupsSection, "{providers_tags}", strings.Join(tags, ", "))
	for _, e := range entries {
		groupsOut += "\n  - {name: " + e.Tag + ", type: url-test, tolerance: 100, lazy: true, use: [" + e.Tag + "]}"
	}
	final := strings.TrimSpace(providerBlock) + "\n" + strings.TrimSpace(groupsOut) + "\n" + strings.TrimSpace(rulesSection) + "\n"

	target := filepath.Join(crashDir, "yamls", "config.yaml")
	return os.WriteFile(target, []byte(final), 0o644)
}

func providersFromArgsOrConfig(crashDir string, args []string) ([]providerGenerateEntry, []string, error) {
	if len(args) > 0 {
		e, err := parseProviderArgs(args)
		if err != nil {
			return nil, nil, err
		}
		return []providerGenerateEntry{e}, []string{e.Tag}, nil
	}

	entries := make([]providerGenerateEntry, 0, 16)
	tags := make([]string, 0, 16)

	cfgRows, err := readLinesIfExists(filepath.Join(crashDir, "configs", "providers.cfg"))
	if err != nil {
		return nil, nil, err
	}
	for _, row := range cfgRows {
		fields := strings.Fields(row)
		if len(fields) < 2 {
			continue
		}
		e, err := parseProviderArgs(fields)
		if err != nil {
			continue
		}
		entries = append(entries, e)
		tags = append(tags, e.Tag)
	}

	uriRows, err := readLinesIfExists(filepath.Join(crashDir, "configs", "providers_uri.cfg"))
	if err != nil {
		return nil, nil, err
	}
	if len(uriRows) > 0 {
		if err := os.MkdirAll(filepath.Join(crashDir, "providers"), 0o755); err != nil {
			return nil, nil, err
		}
		uriGroup := filepath.Join(crashDir, "providers", "uri_group")
		uriItems := make([]string, 0, len(uriRows))
		for _, row := range uriRows {
			fields := strings.Fields(row)
			if len(fields) < 2 {
				continue
			}
			entry := fields[1]
			if fields[0] != "vmess" {
				entry += "#" + fields[0]
			}
			uriItems = append(uriItems, entry)
		}
		if len(uriItems) > 0 {
			if err := os.WriteFile(uriGroup, []byte(strings.Join(uriItems, "\n")+"\n"), 0o644); err != nil {
				return nil, nil, err
			}
			entries = append(entries, providerGenerateEntry{Tag: "Uri_group", Link: "./providers/uri_group", Interval: 3, Interval2: 12, UA: "clash.meta", IsURI: true})
			tags = append(tags, "Uri_group")
		}
	}

	return entries, tags, nil
}

func parseProviderArgs(args []string) (providerGenerateEntry, error) {
	if len(args) < 2 {
		return providerGenerateEntry{}, fmt.Errorf("provider name/link required")
	}
	e := providerGenerateEntry{Tag: args[0], Link: args[1], Interval: 3, Interval2: 12, UA: "clash.meta"}
	if len(args) > 2 {
		if v, err := strconv.Atoi(args[2]); err == nil && v > 0 {
			e.Interval = v
		}
	}
	if len(args) > 3 {
		if v, err := strconv.Atoi(args[3]); err == nil && v > 0 {
			e.Interval2 = v
		}
	}
	if len(args) > 4 && strings.TrimSpace(args[4]) != "" {
		e.UA = args[4]
	}
	if len(args) > 5 {
		e.Exclude = strings.TrimPrefix(args[5], "#")
	}
	if len(args) > 6 {
		e.Include = strings.TrimPrefix(args[6], "#")
	}
	return e, nil
}

func resolveProviderTemplatePath(crashDir, coreType string, cfg map[string]string) (string, error) {
	key := "provider_temp_" + coreType
	name := strings.TrimSpace(stripQuotes(cfg[key]))
	if name == "" {
		entries, err := loadProviderTemplateEntries(crashDir, coreType)
		if err != nil {
			return "", err
		}
		name = entries[0].File
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	local := filepath.Join(crashDir, name)
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}
	if _, err := os.Stat(name); err == nil {
		return name, nil
	}
	rel := filepath.Join(crashDir, "rules", coreType+"_providers", name)
	if _, err := os.Stat(rel); err == nil {
		return rel, nil
	}
	return "", fmt.Errorf("provider template not found: %s", name)
}

func splitClashProviderTemplate(template string) (string, string, error) {
	lines := strings.Split(template, "\n")
	idxGroups := -1
	idxRules := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if idxGroups < 0 && strings.HasPrefix(trimmed, "proxy-groups:") {
			idxGroups = i
		}
		if idxRules < 0 && strings.HasPrefix(trimmed, "rules:") {
			idxRules = i
		}
	}
	if idxGroups < 0 || idxRules < 0 || idxRules <= idxGroups {
		return "", "", fmt.Errorf("invalid clash provider template format")
	}
	groups := strings.Join(lines[idxGroups:idxRules], "\n")
	rules := strings.Join(lines[idxRules:], "\n")
	return strings.TrimSpace(groups), strings.TrimSpace(rules), nil
}

func buildClashProvidersBlock(entries []providerGenerateEntry, cfg map[string]string) string {
	var b strings.Builder
	b.WriteString("proxy-providers:\n")
	crashcore := stripQuotes(cfg["crashcore"])
	skipCert := !strings.EqualFold(stripQuotes(cfg["skip_cert"]), "OFF")
	for _, e := range entries {
		typeValue := "http"
		pathValue := "./providers/" + e.Tag + ".yaml"
		urlValue := e.Link
		if strings.HasPrefix(e.Link, "./") {
			typeValue = "file"
			pathValue = e.Link
			urlValue = ""
		}
		b.WriteString("  " + e.Tag + ":\n")
		b.WriteString("    type: " + typeValue + "\n")
		b.WriteString("    url: \"" + urlValue + "\"\n")
		b.WriteString("    path: \"" + pathValue + "\"\n")
		b.WriteString("    interval: " + strconv.Itoa(e.Interval2*3600) + "\n")
		b.WriteString("    health-check:\n")
		b.WriteString("      enable: true\n")
		b.WriteString("      lazy: true\n")
		b.WriteString("      url: \"https://www.gstatic.com/generate_204\"\n")
		b.WriteString("      interval: " + strconv.Itoa(e.Interval*60) + "\n")
		if crashcore == "meta" {
			b.WriteString("    header:\n")
			b.WriteString("      User-Agent: [\"" + e.UA + "\"]\n")
			b.WriteString("    override:\n")
			b.WriteString("      udp: true\n")
			if skipCert {
				b.WriteString("      skip-cert-verify: true\n")
			}
			b.WriteString("    filter: \"" + e.Include + "\"\n")
			b.WriteString("    exclude-filter: \"" + e.Exclude + "\"\n")
		}
	}
	return b.String()
}
