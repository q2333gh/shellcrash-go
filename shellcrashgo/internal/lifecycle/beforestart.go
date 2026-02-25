package lifecycle

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type BeforeStartOptions struct {
	CrashDir   string
	BinDir     string
	TmpDir     string
	CoreConfig string
	Host       string
	MixPort    string
	URL        string
	HTTPS      string
}

type BeforeStartDeps struct {
	EnsureCoreConfig func() error
	RunTaskScript    func(path string) error
	DetectHost       func() string
}

func BeforeStart(opts BeforeStartOptions, deps BeforeStartDeps) error {
	if opts.CrashDir == "" || opts.TmpDir == "" || opts.CoreConfig == "" {
		return fmt.Errorf("CrashDir/TmpDir/CoreConfig required")
	}
	if opts.BinDir == "" {
		opts.BinDir = opts.CrashDir
	}
	if opts.MixPort == "" {
		opts.MixPort = "7890"
	}
	if deps.EnsureCoreConfig == nil {
		return fmt.Errorf("EnsureCoreConfig dependency is required")
	}
	if deps.RunTaskScript == nil {
		deps.RunTaskScript = func(string) error { return nil }
	}
	if deps.DetectHost == nil {
		deps.DetectHost = detectHostFromInterfaces
	}

	if err := os.MkdirAll(opts.TmpDir, 0o755); err != nil {
		return err
	}
	if fileExists(filepath.Join(opts.CrashDir, ".start_error")) {
		return fmt.Errorf("last startup failed")
	}

	task := filepath.Join(opts.CrashDir, "task", "bfstart")
	if stat, err := os.Stat(task); err == nil && stat.Size() > 0 {
		if err := deps.RunTaskScript(task); err != nil {
			return err
		}
	}

	if !fileExists(opts.CoreConfig) {
		if opts.URL == "" && opts.HTTPS == "" {
			return fmt.Errorf("core config link missing, import config first")
		}
		if err := deps.EnsureCoreConfig(); err != nil {
			return err
		}
	}

	binUI := filepath.Join(opts.BinDir, "ui")
	if err := os.MkdirAll(binUI, 0o755); err != nil {
		return err
	}
	uiCNAME := filepath.Join(opts.CrashDir, "ui", "CNAME")
	if fileExists(uiCNAME) && !fileExists(filepath.Join(binUI, "CNAME")) {
		if err := copyDir(filepath.Join(opts.CrashDir, "ui"), binUI); err != nil {
			return err
		}
	}
	indexPath := filepath.Join(binUI, "index.html")
	if !fileExists(indexPath) || fileSize(indexPath) == 0 {
		if err := os.WriteFile(indexPath, []byte(defaultUIPromptHTML), 0o644); err != nil {
			return err
		}
	}

	host := strings.TrimSpace(opts.Host)
	if host == "" {
		host = deps.DetectHost()
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if err := writePAC(filepath.Join(binUI, "pac"), host, opts.MixPort); err != nil {
		return err
	}

	if opts.CrashDir != opts.BinDir {
		if err := linkProviders(opts.CrashDir, opts.BinDir); err != nil {
			return err
		}
	}

	_ = os.Remove(filepath.Join("/tmp/ShellCrash", "debug.log"))
	_ = os.Remove(filepath.Join(opts.CrashDir, "debug.log"))
	_ = os.Remove(filepath.Join(opts.TmpDir, "debug.log"))
	return nil
}

func detectHostFromInterfaces() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			v4 := ip.To4()
			if v4 == nil || v4.IsLoopback() {
				continue
			}
			if isPrivateIPv4(v4) {
				return v4.String()
			}
		}
	}
	return ""
}

func isPrivateIPv4(ip net.IP) bool {
	if len(ip) != net.IPv4len {
		return false
	}
	switch {
	case ip[0] == 10:
		return true
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return true
	case ip[0] == 192 && ip[1] == 168:
		return true
	default:
		return false
	}
}

func copyDir(srcDir string, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return err
			}
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(src)
			if err != nil {
				return err
			}
			_ = os.RemoveAll(dst)
			if err := os.Symlink(target, dst); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(src, dst, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src string, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func writePAC(path string, host string, mixPort string) error {
	content := fmt.Sprintf(`function FindProxyForURL(url, host) {
	if (
		isInNet(host, "0.0.0.0", "255.0.0.0")||
		isInNet(host, "10.0.0.0", "255.0.0.0")||
		isInNet(host, "127.0.0.0", "255.0.0.0")||
		isInNet(host, "224.0.0.0", "224.0.0.0")||
		isInNet(host, "240.0.0.0", "240.0.0.0")||
		isInNet(host, "172.16.0.0",  "255.240.0.0")||
		isInNet(host, "192.168.0.0", "255.255.0.0")||
		isInNet(host, "169.254.0.0", "255.255.0.0")
	)
		return "DIRECT";
	else
		return "PROXY %s:%s; DIRECT; SOCKS5 %s:%s"
}
`, host, mixPort, host, mixPort)

	if b, err := os.ReadFile(path); err == nil && bytes.Equal(b, []byte(content)) {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func linkProviders(crashDir string, binDir string) error {
	srcDir := filepath.Join(crashDir, "providers")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dstDir := filepath.Join(binDir, "providers")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		_ = os.RemoveAll(dst)
		if err := os.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

const defaultUIPromptHTML = `<!DOCTYPE html>
<html lang="en">
<meta http-equiv="Cache-Control" content="no-cache, no-store, must-revalidate">
<meta http-equiv="Pragma" content="no-cache">
<meta http-equiv="Expires" content="0">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ShellCrash面板提示</title>
</head>
<body>
    <div style="text-align: center; margin-top: 50px;">
        <h1>您还未安装本地面板</h1>
		<h3>请在脚本更新功能中(9-4)安装<br>或者使用在线面板：</h3>
		<h4>请复制当前地址/ui(不包括)前面的内容，填入url位置即可连接</h3>
        <a href="http://board.zash.run.place" style="font-size: 24px;">Zashboard面板(推荐)<br></a>
        <a style="font-size: 21px;"><br>如已安装，请使用Ctrl+F5强制刷新此页面！<br></a>
    </div>
</body>
</html
`
