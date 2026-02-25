package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// WebGetOptions configures web download behavior
type WebGetOptions struct {
	ProxyURL        string // HTTP/HTTPS proxy URL
	UserAgent       string // Custom user agent
	SkipCertCheck   bool   // Skip certificate verification
	NoRedirect      bool   // Disable following redirects
	Timeout         int    // Timeout in seconds (default: 30)
	ShowProgress    bool   // Show download progress
	RewriteGitHub   bool   // Rewrite GitHub URLs when proxy is active
}

// WebGet downloads a file from a URL to a local path
// Returns error if download fails
func WebGet(localPath, remoteURL string, opts *WebGetOptions) error {
	if opts == nil {
		opts = &WebGetOptions{}
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30
	}

	// Rewrite URL if proxy is active and RewriteGitHub is enabled
	finalURL := remoteURL
	if opts.RewriteGitHub && opts.ProxyURL != "" {
		finalURL = rewriteGitHubURL(remoteURL, true)
	} else if !opts.RewriteGitHub || opts.ProxyURL == "" {
		finalURL = rewriteGitHubURL(remoteURL, false)
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: time.Duration(opts.Timeout) * time.Second,
	}

	// Configure proxy
	if opts.ProxyURL != "" {
		proxyURL, err := url.Parse(opts.ProxyURL)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	// Configure redirect policy
	if opts.NoRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Create request
	req, err := http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	} else {
		req.Header.Set("User-Agent", "ShellCrash/1.0")
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		// Retry without proxy if first attempt fails
		if opts.ProxyURL != "" {
			client.Transport = nil
			resp, err = client.Do(req)
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}
		} else {
			return fmt.Errorf("download failed: %w", err)
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// rewriteGitHubURL rewrites GitHub URLs based on proxy status
// When proxy is active: jsdelivr -> raw.githubusercontent.com
// When proxy is inactive: raw.githubusercontent.com -> jsdelivr
func rewriteGitHubURL(urlStr string, proxyActive bool) string {
	if proxyActive {
		// When proxy is active, use raw.githubusercontent.com
		urlStr = strings.Replace(urlStr,
			"https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@",
			"https://raw.githubusercontent.com/juewuy/ShellCrash/", 1)
		urlStr = strings.Replace(urlStr,
			"https://gh.jwsc.eu.org/",
			"https://raw.githubusercontent.com/juewuy/ShellCrash/", 1)
	} else {
		// When proxy is inactive, use jsdelivr CDN
		urlStr = strings.Replace(urlStr,
			"https://raw.githubusercontent.com/juewuy/ShellCrash/",
			"https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@", 1)
	}
	return urlStr
}

// GetBinOptions configures project-specific binary download
type GetBinOptions struct {
	UpdateURL    string // Base update URL
	URLId        string // Server ID from configs/servers.list
	ReleaseType  string // Release type (master, dev, update)
	WebGetOpts   *WebGetOptions
}

// GetBin downloads project-specific files using configured servers
func GetBin(localPath, remotePath, crashDir string, opts *GetBinOptions) error {
	if opts == nil {
		opts = &GetBinOptions{}
	}
	if opts.UpdateURL == "" {
		opts.UpdateURL = "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master"
	}

	var binURL string
	if opts.URLId != "" {
		// Determine release type based on path
		rt := opts.ReleaseType
		if rt == "" {
			rt = "master"
		}
		if strings.HasPrefix(remotePath, "bin/") {
			rt = "update"
		} else if strings.HasPrefix(remotePath, "public/") || strings.HasPrefix(remotePath, "rules/") {
			rt = "dev"
		}

		// Read server URL from configs/servers.list
		serversFile := crashDir + "/configs/servers.list"
		if data, err := os.ReadFile(serversFile); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 3 && fields[0] == opts.URLId {
					baseURL := fields[2]
					// Special handling for jsdelivr
					if opts.URLId == "101" || opts.URLId == "104" {
						binURL = baseURL + "@" + rt + "/" + remotePath
					} else {
						binURL = baseURL + "/" + rt + "/" + remotePath
					}
					break
				}
			}
		}
	}

	if binURL == "" {
		binURL = opts.UpdateURL + "/" + remotePath
	}

	return WebGet(localPath, binURL, opts.WebGetOpts)
}
