package startctl

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var coreArchiveNamePattern = regexp.MustCompile(`(?i)(CrashCore|sing|meta|mihomo|clash|pre)`)
var singboxOutboundPattern = regexp.MustCompile(`(?i)"(socks|http|shadowsocksr?|vmess|trojan|wireguard|hysteria2?|vless|shadowtls|tuic|ssh|tor|providers|anytls|soduku)"`)
var singboxLegacyDNSOutPattern = regexp.MustCompile(`\{[^{}]*"dns-out"[^{}]*}`)
var singboxLegacyPrefixPattern = regexp.MustCompile(`(?s)^.*"inbounds":`)

func (c *Controller) runCorePreChecks(coreConfig string) error {
	if err := c.validateCoreConfigContent(coreConfig); err != nil {
		return err
	}
	if err := c.checkConfigCoreCompatibility(coreConfig); err != nil {
		return err
	}
	if err := c.runGeoPreChecks(coreConfig); err != nil {
		return err
	}
	if err := c.ensureCrashCoreBinary(); err != nil {
		return err
	}
	return c.prepareRuntimeConfig(coreConfig)
}

func (c *Controller) validateCoreConfigContent(coreConfig string) error {
	raw, err := os.ReadFile(coreConfig)
	if err != nil {
		return err
	}
	text := string(raw)
	if strings.Contains(c.Cfg.CrashCore, "singbox") {
		return validateSingboxConfigContent(coreConfig, text)
	}
	return validateClashConfigContent(coreConfig, text)
}

func validateClashConfigContent(path, text string) error {
	requiresNodeValidation := strings.Contains(text, "proxies:") || strings.Contains(text, "proxy-providers:")
	if requiresNodeValidation {
		proxiesSection := extractClashProxiesSection(text)
		if !strings.Contains(proxiesSection, "server:") &&
			!strings.Contains(proxiesSection, `server":`) &&
			!strings.Contains(proxiesSection, "server':") &&
			!hasExplicitEmptyClashProxies(text) &&
			!strings.Contains(text, "proxy-providers:") {
			return fmt.Errorf("config %s has no valid clash nodes/providers", path)
		}
	}
	if strings.Contains(text, "Proxy Group:") {
		return fmt.Errorf("config %s uses unsupported legacy clash format", path)
	}
	if strings.Contains(text, "cipher: chacha20,") {
		return fmt.Errorf("config %s uses unsupported cipher chacha20", path)
	}
	return nil
}

func extractClashProxiesSection(text string) string {
	lines := strings.Split(text, "\n")
	var b strings.Builder
	inProxies := false
	for _, line := range lines {
		if !inProxies {
			if strings.TrimSpace(line) == "proxies:" {
				inProxies = true
			}
			continue
		}
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") &&
			!strings.HasPrefix(trimmed, "-") && strings.Contains(trimmed, ":") {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func hasExplicitEmptyClashProxies(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "proxies: []" {
			return true
		}
	}
	return false
}

func validateSingboxConfigContent(path, text string) error {
	normalized := text
	if countLines(text) < 3 {
		normalized = singboxLegacyPrefixPattern.ReplaceAllString(normalized, `{"inbounds":`)
		normalized = singboxLegacyDNSOutPattern.ReplaceAllString(normalized, "")
		normalized = strings.ReplaceAll(normalized, ",,", ",")
		if normalized != text {
			if err := os.WriteFile(path, []byte(normalized), 0o644); err != nil {
				return err
			}
			text = normalized
		}
	}
	if !singboxOutboundPattern.MatchString(text) {
		return fmt.Errorf("config %s has no valid sing-box nodes/providers", path)
	}
	if strings.Contains(text, `"sni"`) {
		return fmt.Errorf("config %s is unsupported legacy sing-box (<1.12) format", path)
	}
	return nil
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func (c *Controller) checkConfigCoreCompatibility(coreConfig string) error {
	content, err := os.ReadFile(coreConfig)
	if err != nil {
		return err
	}
	text := string(content)

	if strings.Contains(c.Cfg.CrashCore, "singbox") {
		if c.Cfg.CrashCore != "singboxr" && (strings.Contains(text, `"shadowsocksr"`) || strings.Contains(text, `"providers"`)) {
			return c.switchCrashCore("singboxr")
		}
		return nil
	}

	if c.Cfg.CrashCore != "meta" && (strings.Contains(text, "type: vless") || strings.Contains(text, "type: hysteria")) {
		return c.switchCrashCore("meta")
	}

	if c.Cfg.CrashCore == "clash" {
		if strings.Contains(strings.ToLower(text), "script:") || strings.Contains(strings.ToLower(text), "proxy-providers") || strings.Contains(strings.ToLower(text), "rule-providers") || strings.Contains(strings.ToLower(text), "rule-set") || strings.EqualFold(c.Cfg.RedirMod, "mix") || strings.EqualFold(c.Cfg.RedirMod, "tun") {
			return c.switchCrashCore("meta")
		}
		if (c.Cfg.FirewallArea == "2" || c.Cfg.FirewallArea == "3") && !passwdHasUID0ShellCrash() {
			return c.switchCrashCore("meta")
		}
	}
	return nil
}

func passwdHasUID0ShellCrash() bool {
	b, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "0:7890")
}

func (c *Controller) switchCrashCore(target string) error {
	if target == "" || c.Cfg.CrashCore == target {
		return nil
	}
	c.Cfg.CrashCore = target
	c.Cfg.Command = defaultCommandForCore(target)

	for _, p := range []string{
		filepath.Join(c.Cfg.TmpDir, "CrashCore"),
		filepath.Join(c.Cfg.BinDir, "CrashCore"),
		filepath.Join(c.Cfg.BinDir, "CrashCore.tar.gz"),
		filepath.Join(c.Cfg.BinDir, "CrashCore.gz"),
		filepath.Join(c.Cfg.BinDir, "CrashCore.upx"),
	} {
		_ = os.RemoveAll(p)
	}

	cfgPath := filepath.Join(c.Cfg.CrashDir, "configs", "ShellCrash.cfg")
	cfgKV, err := parseKVFile(cfgPath)
	if err == nil {
		cfgKV["crashcore"] = target
		if err := writeKVFile(cfgPath, cfgKV); err != nil {
			return err
		}
	}

	envPath := filepath.Join(c.Cfg.CrashDir, "configs", "command.env")
	envKV, err := parseKVFile(envPath)
	if err == nil {
		envKV["COMMAND"] = "'" + c.Cfg.Command + "'"
		if err := writeKVFile(envPath, envKV); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureCrashCoreBinary() error {
	target := filepath.Join(c.Cfg.TmpDir, "CrashCore")
	if isUsableCoreBinary(target) {
		return nil
	}
	if err := os.MkdirAll(c.Cfg.TmpDir, 0o755); err != nil {
		return err
	}

	for _, src := range c.coreCandidates() {
		if err := unpackCoreCandidate(src, target); err == nil && isUsableCoreBinary(target) {
			return nil
		}
	}
	return fmt.Errorf("CrashCore not found; download/install core first")
}

func (c *Controller) coreCandidates() []string {
	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	dirs := []string{c.Cfg.BinDir}
	if c.Cfg.CrashDir != c.Cfg.BinDir {
		dirs = append(dirs, c.Cfg.CrashDir)
	}
	for _, dir := range dirs {
		for _, name := range []string{"CrashCore", "CrashCore.tar.gz", "CrashCore.gz", "CrashCore.upx"} {
			p := filepath.Join(dir, name)
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

func unpackCoreCandidate(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil || info.Size() <= 2000 {
		return fmt.Errorf("skip")
	}

	switch {
	case strings.HasSuffix(src, ".tar.gz"):
		return extractFromTarGz(src, dst)
	case strings.HasSuffix(src, ".gz"):
		return extractFromGzip(src, dst)
	case strings.HasSuffix(src, ".upx"):
		_ = os.RemoveAll(dst)
		return os.Symlink(src, dst)
	default:
		if err := copyFileWithPerm(src, dst, 0o755); err != nil {
			return err
		}
		return os.Chmod(dst, 0o755)
	}
}

func copyFileWithPerm(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func extractFromGzip(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	zr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer zr.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, zr); err != nil {
		return err
	}
	return out.Sync()
}

func extractFromTarGz(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	zr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || hdr.Size <= 2000 {
			continue
		}
		name := filepath.Base(hdr.Name)
		if !coreArchiveNamePattern.MatchString(name) {
			continue
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		return os.Chmod(dst, 0o755)
	}
	return fmt.Errorf("no suitable core binary in tar")
}

func isUsableCoreBinary(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 2000
}
