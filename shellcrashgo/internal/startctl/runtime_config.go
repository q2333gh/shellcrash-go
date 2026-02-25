package startctl

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (c *Controller) prepareRuntimeConfig(coreConfig string) error {
	if strings.Contains(c.Cfg.CrashCore, "singbox") {
		return c.prepareSingboxRuntimeConfig(coreConfig)
	}
	return c.prepareClashRuntimeConfig(coreConfig)
}

func (c *Controller) prepareClashRuntimeConfig(coreConfig string) error {
	target := filepath.Join(c.Cfg.TmpDir, "config.yaml")
	if err := os.MkdirAll(c.Cfg.TmpDir, 0o755); err != nil {
		return err
	}
	if c.Cfg.DisOverride == "1" {
		if err := copyFileWithMode(coreConfig, target, 0o644); err != nil {
			return err
		}
	} else {
		raw, err := os.ReadFile(coreConfig)
		if err != nil {
			return err
		}
		out, err := c.buildClashRuntimeConfigFromCore(string(raw), true)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, out, 0o644); err != nil {
			return err
		}
		if err := c.validateClashRuntimeConfig(target); err != nil {
			out, buildErr := c.buildClashRuntimeConfigFromCore(string(raw), false)
			if buildErr != nil {
				return buildErr
			}
			if writeErr := os.WriteFile(target, out, 0o644); writeErr != nil {
				return writeErr
			}
		}
	}

	// Keep legacy path for dashboards/tools that still read from BINDIR.
	if c.Cfg.BinDir != "" {
		binTarget := filepath.Join(c.Cfg.BinDir, "config.yaml")
		if err := os.MkdirAll(c.Cfg.BinDir, 0o755); err == nil {
			_ = os.RemoveAll(binTarget)
			if err := os.Symlink(target, binTarget); err != nil {
				_ = copyFileWithMode(target, binTarget, 0o644)
			}
		}
	}
	return nil
}

func (c *Controller) validateClashRuntimeConfig(path string) error {
	core := filepath.Join(c.Cfg.TmpDir, "CrashCore")
	info, err := os.Stat(core)
	if err != nil || info.IsDir() || info.Size() <= 2000 {
		return nil
	}
	cmd := exec.Command(core, "-t", "-d", c.Cfg.BinDir, "-f", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.WriteFile(filepath.Join(c.Cfg.TmpDir, "error.yaml"), mustReadFile(path), 0o644)
		return fmt.Errorf("clash runtime validate failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func mustReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func (c *Controller) prepareSingboxRuntimeConfig(coreConfig string) error {
	jsonDir := filepath.Join(c.Cfg.TmpDir, "jsons")
	if err := os.RemoveAll(jsonDir); err != nil {
		return err
	}
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		return err
	}

	base := filepath.Join(jsonDir, "00_config.json")
	if err := copyFileWithMode(coreConfig, base, 0o644); err != nil {
		return err
	}
	if err := c.writeSingboxRuntimeOverlays(jsonDir, base); err != nil {
		return err
	}
	legacy := filepath.Join(jsonDir, "config.json")
	_ = os.RemoveAll(legacy)
	if err := os.Symlink(base, legacy); err != nil {
		if err := copyFileWithMode(base, legacy, 0o644); err != nil {
			return err
		}
	}
	if err := c.enforceSingboxSkipCert(jsonDir); err != nil {
		return err
	}

	if c.Cfg.DisOverride == "1" {
		return nil
	}

	customDir := filepath.Join(c.Cfg.CrashDir, "jsons")
	entries, err := os.ReadDir(customDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		src := filepath.Join(customDir, entry.Name())
		if sameFile(src, coreConfig) {
			continue
		}
		dst := filepath.Join(jsonDir, "50_"+entry.Name())
		if err := copyFileWithMode(src, dst, 0o644); err != nil {
			return fmt.Errorf("copy custom json %s: %w", entry.Name(), err)
		}
	}
	if err := c.enforceSingboxSkipCert(jsonDir); err != nil {
		return err
	}
	return nil
}

func copyFileWithMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func sameFile(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return aa == bb
}
