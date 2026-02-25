package startctl

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var yamlPatternGeodataMode = regexp.MustCompile(`(?m)^[ \t]*geodata-mode:`)
var yamlPatternGeosite = regexp.MustCompile(`(?m)^[ \t]*geosite:`)
var yamlPatternCNRuleProvider = regexp.MustCompile(`(?m)^[ \t]*cn:`)
var jsonPatternCNRuleSet = regexp.MustCompile(`"tag"\s*:\s*"cn"`)

type geoAssetDownloader func(dst string, remoteName string) error

var startctlGeoDownloader geoAssetDownloader
var startctlRunIPSetRestore = runIPSetRestore
var startctlDestroyIPSet = destroyIPSet

func (c *Controller) runGeoPreChecks(coreConfig string) error {
	if err := c.ensureCoreGeoAssets(coreConfig); err != nil {
		return err
	}
	return c.ensureCNIPAssets()
}

func (c *Controller) ensureCoreGeoAssets(coreConfig string) error {
	if strings.Contains(c.Cfg.CrashCore, "singbox") {
		if !isDNSModeMixOrRoute(c.Cfg.DNSMod) {
			return nil
		}
		hasCN, err := anyJSONFileMatches(filepath.Join(c.Cfg.CrashDir, "jsons"), jsonPatternCNRuleSet)
		if err != nil {
			return err
		}
		if !hasCN {
			return c.ensureGeoFile("ruleset/cn.srs", "srs_geosite_cn.srs")
		}
		return nil
	}

	content, err := os.ReadFile(coreConfig)
	if err != nil {
		return err
	}
	text := strings.ToUpper(string(content))

	geodataModeSet, err := anyYAMLFileMatches(filepath.Join(c.Cfg.CrashDir, "yamls"), yamlPatternGeodataMode)
	if err != nil {
		return err
	}
	if strings.Contains(text, "GEOIP,CN") && !geodataModeSet {
		if err := c.ensureGeoFile("Country.mmdb", "cn_mini.mmdb"); err != nil {
			return err
		}
	}

	geositeSet, err := anyYAMLFileMatches(filepath.Join(c.Cfg.CrashDir, "yamls"), yamlPatternGeosite)
	if err != nil {
		return err
	}
	if strings.Contains(text, "GEOSITE,") && !geositeSet {
		if err := c.ensureGeoFile("GeoSite.dat", "geosite.dat"); err != nil {
			return err
		}
	}

	if isDNSModeMixOrRoute(c.Cfg.DNSMod) {
		hasCN, err := anyYAMLFileMatches(filepath.Join(c.Cfg.CrashDir, "yamls"), yamlPatternCNRuleProvider)
		if err != nil {
			return err
		}
		if !hasCN {
			if err := c.ensureGeoFile("ruleset/cn.mrs", "mrs_geosite_cn.mrs"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) ensureCNIPAssets() error {
	if !strings.EqualFold(c.Cfg.CNIPRoute, "ON") || strings.EqualFold(c.Cfg.DNSMod, "fake-ip") {
		return nil
	}
	return c.ensureCNIPAssetsForMode("", true)
}

func (c *Controller) ensureCNIPAssetsForMode(mode string, respectRouteGate bool) error {
	if respectRouteGate && (!strings.EqualFold(c.Cfg.CNIPRoute, "ON") || strings.EqualFold(c.Cfg.DNSMod, "fake-ip")) {
		return nil
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "v4" {
		mode = "ipv4"
	}
	if mode == "v6" {
		mode = "ipv6"
	}
	if mode != "" && mode != "ipv4" && mode != "ipv6" {
		return fmt.Errorf("unsupported check_cnip mode: %s", mode)
	}

	if c.Cfg.FirewallMod != "nftables" && !hasCommand("ipset") {
		return nil
	}
	checkIPv4 := mode == "" || mode == "ipv4"
	checkIPv6 := mode == "ipv6" || (mode == "" && strings.EqualFold(c.Cfg.IPv6Redir, "ON"))

	if checkIPv4 {
		if err := c.ensureGeoFile("cn_ip.txt", "china_ip_list.txt"); err != nil {
			return err
		}
	}
	if checkIPv6 {
		if err := c.ensureGeoFile("cn_ipv6.txt", "china_ipv6_list.txt"); err != nil {
			return err
		}
	}
	if c.Cfg.FirewallMod != "iptables" || !hasCommand("ipset") {
		return nil
	}
	if checkIPv4 {
		if err := restoreIPSetFromCIDRFile(filepath.Join(c.Cfg.BinDir, "cn_ip.txt"), "cn_ip", "inet", "10240", startctlDestroyIPSet, startctlRunIPSetRestore); err != nil {
			return err
		}
	}
	if checkIPv6 {
		if err := restoreIPSetFromCIDRFile(filepath.Join(c.Cfg.BinDir, "cn_ipv6.txt"), "cn_ip6", "inet6", "5120", startctlDestroyIPSet, startctlRunIPSetRestore); err != nil {
			return err
		}
	}
	return nil
}

func isDNSModeMixOrRoute(mode string) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	return mode == "mix" || mode == "route"
}

func (c *Controller) ensureGeoFile(relPath string, remoteName string) error {
	dst := filepath.Join(c.Cfg.BinDir, filepath.FromSlash(relPath))
	if isLargeFile(dst, 20) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	src := filepath.Join(c.Cfg.CrashDir, filepath.FromSlash(relPath))
	if isLargeFile(src, 20) {
		_ = os.RemoveAll(dst)
		if err := os.Rename(src, dst); err == nil {
			return nil
		}
		if err := copyFileWithPerm(src, dst, 0o644); err == nil && isLargeFile(dst, 20) {
			return nil
		}
	}

	downloader := startctlGeoDownloader
	if downloader == nil {
		downloader = func(dst string, remoteName string) error {
			return downloadGeoAsset(dst, remoteName)
		}
	}
	if err := downloader(dst, remoteName); err != nil {
		return err
	}
	if !isLargeFile(dst, 20) {
		return fmt.Errorf("%s missing or invalid after download", relPath)
	}
	return nil
}

func downloadGeoAsset(dst string, remoteName string) error {
	baseURL := "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@update/bin/geodata/"
	req, err := http.NewRequest(http.MethodGet, baseURL+remoteName, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s failed: http %d", remoteName, resp.StatusCode)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return out.Sync()
}

func anyYAMLFileMatches(dir string, re *regexp.Regexp) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return false, err
		}
		if re.Match(b) {
			return true, nil
		}
	}
	return false, nil
}

func anyJSONFileMatches(dir string, re *regexp.Regexp) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return false, err
		}
		if re.Match(b) {
			return true, nil
		}
	}
	return false, nil
}

func isLargeFile(path string, minBytes int64) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > minBytes
}

func restoreIPSetFromCIDRFile(cidrFile, setName, family, maxElem string, destroy func(string), restore func(string) error) error {
	b, err := os.ReadFile(cidrFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(b), "\n")
	commands := []string{
		fmt.Sprintf("create %s hash:net family %s hashsize %s maxelem %s", setName, family, maxElem, maxElem),
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		commands = append(commands, fmt.Sprintf("add %s %s", setName, line))
	}
	tmp, err := os.CreateTemp("", "shellcrash-ipset-*.restore")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(strings.Join(commands, "\n") + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if destroy != nil {
		destroy(setName)
	}
	if restore == nil {
		return nil
	}
	return restore(tmpPath)
}

func destroyIPSet(setName string) {
	_ = exec.Command("ipset", "destroy", setName).Run()
}

func runIPSetRestore(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd := exec.Command("ipset", "-!", "restore")
	cmd.Stdin = f
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
