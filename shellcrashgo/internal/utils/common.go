package utils

import (
	"net/url"
	"os/exec"
)

// URLEncode encodes a string for use in URLs
func URLEncode(s string) string {
	return url.QueryEscape(s)
}

// URLDecode decodes a URL-encoded string
func URLDecode(s string) (string, error) {
	return url.QueryUnescape(s)
}

// CommandExists checks if a command is available in PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
