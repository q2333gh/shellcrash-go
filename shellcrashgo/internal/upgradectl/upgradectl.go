package upgradectl

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"shellcrash/internal/taskctl"
	"shellcrash/internal/uninstallctl"
)

type Options struct {
	CrashDir       string
	OpenSSLDir     string
	CertSourcePath string
	BinDir         string
}

type State struct {
	ReleaseType string
	URLID       string
	UpdateURL   string
}

type ServerEntry struct {
	ID   int
	Name string
	URL  string
	Raw  string
}

var upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	return client.Do(req)
}

var upgradeExecOutput = func(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

var upgradeRunTask = func(crashDir, action string) error {
	r := taskctl.Runner{CrashDir: crashDir}
	return r.Run([]string{action})
}

var upgradeRunSetScriptsMenu = RunSetScriptsMenu
var upgradeRunSetCoreMenu = RunSetCoreMenu
var upgradeRunSetGeoMenu = RunSetGeoMenu
var upgradeRunSetDBMenu = RunSetDBMenu
var upgradeRunSetCertMenu = RunSetCertMenu
var upgradeRunSetServerMenu = RunSetServerMenu
var upgradeRunUninstallMenu = func(opts uninstallctl.Options, in io.Reader, out io.Writer) error {
	return uninstallctl.RunMenu(opts, uninstallctl.Deps{}, in, out)
}

type CoreState struct {
	CrashCore string
	CPUCore   string
	ZipType   string
	CoreV     string
}

func RunUpgradeMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintln(out, "更新与支持")
		fmt.Fprintln(out, "1) 更新管理脚本")
		fmt.Fprintln(out, "2) 切换/更新内核文件")
		fmt.Fprintln(out, "3) 安装/更新本地数据库文件")
		fmt.Fprintln(out, "4) 安装/更新本地Dashboard面板")
		fmt.Fprintln(out, "5) 安装/更新本地根证书文件")
		fmt.Fprintln(out, "6) PAC自动代理地址")
		fmt.Fprintln(out, "7) 切换安装源及版本分支")
		fmt.Fprintln(out, "8) 卸载ShellCrash")
		fmt.Fprintln(out, "9) 感谢列表")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := upgradeRunSetScriptsMenu(opts, reader, out); err != nil {
				return err
			}
		case "2":
			if err := upgradeRunSetCoreMenu(opts, reader, out); err != nil {
				return err
			}
		case "3":
			if err := upgradeRunSetGeoMenu(opts, reader, out); err != nil {
				return err
			}
		case "4":
			if err := upgradeRunSetDBMenu(opts, reader, out); err != nil {
				return err
			}
		case "5":
			if err := upgradeRunSetCertMenu(opts, reader, out); err != nil {
				return err
			}
		case "6":
			fmt.Fprintf(out, "PAC配置链接：http://%s:%s/ui/pac\n", defaultHost(opts.CrashDir), defaultDBPort(opts.CrashDir))
		case "7":
			if err := upgradeRunSetServerMenu(opts, reader, out); err != nil {
				return err
			}
		case "8":
			if err := upgradeRunUninstallMenu(uninstallctl.Options{
				CrashDir: opts.CrashDir,
				BinDir:   opts.BinDir,
				Alias:    strings.TrimSpace(os.Getenv("my_alias")),
			}, reader, out); err != nil {
				return err
			}
		case "9":
			fmt.Fprintln(out, "感谢所有开源项目及贡献者对 ShellCrash 的支持。")
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunSetCoreMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		st, err := loadCoreState(opts.CrashDir)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "切换/更新内核文件")
		fmt.Fprintf(out, "当前内核: %s %s\n", st.CrashCore, st.CoreV)
		fmt.Fprintf(out, "当前架构: %s\n", st.CPUCore)
		fmt.Fprintf(out, "当前压缩: %s\n", st.ZipType)
		fmt.Fprintln(out, "1) 使用Mihomo内核")
		fmt.Fprintln(out, "2) 使用Singbox-reF1nd内核")
		fmt.Fprintln(out, "3) 使用Singbox内核")
		fmt.Fprintln(out, "4) 使用Clash内核")
		fmt.Fprintln(out, "5) 切换压缩格式")
		fmt.Fprintln(out, "7) 更新当前内核")
		fmt.Fprintln(out, "9) 手动指定处理器架构")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := installCoreAsset(opts, "meta", st.CPUCore, st.ZipType); err != nil {
				return err
			}
			fmt.Fprintln(out, "Mihomo内核更新成功")
			return nil
		case "2":
			if err := installCoreAsset(opts, "singboxr", st.CPUCore, st.ZipType); err != nil {
				return err
			}
			fmt.Fprintln(out, "Singbox-reF1nd内核更新成功")
			return nil
		case "3":
			if err := installCoreAsset(opts, "singbox", st.CPUCore, st.ZipType); err != nil {
				return err
			}
			fmt.Fprintln(out, "Singbox内核更新成功")
			return nil
		case "4":
			if err := installCoreAsset(opts, "clash", st.CPUCore, st.ZipType); err != nil {
				return err
			}
			fmt.Fprintln(out, "Clash内核更新成功")
			return nil
		case "5":
			if err := runZipTypeMenu(opts.CrashDir, reader, out); err != nil {
				return err
			}
		case "7":
			if err := installCoreAsset(opts, st.CrashCore, st.CPUCore, st.ZipType); err != nil {
				return err
			}
			fmt.Fprintln(out, "当前内核更新成功")
			return nil
		case "9":
			if err := runCPUCoreMenu(opts.CrashDir, reader, out); err != nil {
				return err
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunSetServerMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)

	for {
		st, err := loadState(opts)
		if err != nil {
			return err
		}
		entries, err := listServerSources(opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "切换ShellCrash版本及更新源地址")
		fmt.Fprintf(out, "当前版本: %s\n", releaseDisplayName(st.ReleaseType))
		fmt.Fprintf(out, "当前源: %s\n", currentURLName(st, entries))
		for i, e := range entries {
			fmt.Fprintf(out, "%d) %s\n", i+1, e.Name)
		}
		fmt.Fprintln(out, "a) 切换至稳定版-stable")
		fmt.Fprintln(out, "b) 切换至公测版-master")
		fmt.Fprintln(out, "c) 切换至开发版-dev")
		fmt.Fprintln(out, "d) 自定义源地址")
		fmt.Fprintln(out, "e) 版本回退")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}

		saved := false
		switch choice {
		case "", "0":
			return nil
		case "a":
			st.ReleaseType = "stable"
			if strings.TrimSpace(st.URLID) == "" {
				st.URLID = "101"
			}
			saved = true
		case "b":
			st.ReleaseType = "master"
			if strings.TrimSpace(st.URLID) == "" {
				st.URLID = "101"
			}
			saved = true
		case "c":
			fmt.Fprintln(out, "开发版可能存在大量bug，是否确认切换？")
			fmt.Fprintln(out, "1) 确认切换")
			fmt.Fprintln(out, "0) 返回")
			fmt.Fprint(out, "请输入对应标号> ")
			confirm, err := readLine(reader)
			if err != nil {
				return err
			}
			if confirm != "1" {
				continue
			}
			st.ReleaseType = "dev"
			if strings.TrimSpace(st.URLID) == "" {
				st.URLID = "101"
			}
			saved = true
		case "d":
			fmt.Fprint(out, "请输入个人源路径(0返回)> ")
			v, err := readLine(reader)
			if err != nil {
				return err
			}
			v = strings.TrimSpace(v)
			if v == "0" || v == "" {
				continue
			}
			st.UpdateURL = v
			st.URLID = ""
			st.ReleaseType = ""
			saved = true
		case "e":
			n, err := strconv.Atoi(st.URLID)
			if err != nil || n >= 200 || n <= 0 {
				fmt.Fprintln(out, "当前源不支持版本回退")
				continue
			}
			tags, err := fetchRollbackTags()
			if err != nil {
				fmt.Fprintln(out, "版本回退信息获取失败")
				continue
			}
			if len(tags) == 0 {
				fmt.Fprintln(out, "未获取到可回退版本")
				continue
			}
			fmt.Fprintln(out, "请选择想要回退至的具体版本:")
			for i, tag := range tags {
				fmt.Fprintf(out, "%d) %s\n", i+1, tag)
			}
			fmt.Fprintln(out, "0) 返回")
			fmt.Fprint(out, "请输入对应标号> ")
			selRaw, err := readLine(reader)
			if err != nil {
				return err
			}
			if selRaw == "" || selRaw == "0" {
				continue
			}
			sel, err := strconv.Atoi(selRaw)
			if err != nil || sel < 1 || sel > len(tags) {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			st.ReleaseType = tags[sel-1]
			st.UpdateURL = ""
			saved = true
		default:
			sel, err := strconv.Atoi(choice)
			if err != nil || sel < 1 || sel > len(entries) {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			entry := entries[sel-1]
			if entry.ID >= 200 {
				st.UpdateURL = entry.URL
				st.URLID = ""
			} else {
				st.URLID = strconv.Itoa(entry.ID)
				st.UpdateURL = ""
			}
			saved = true
		}
		if saved {
			if err := saveState(opts, st); err != nil {
				return err
			}
			fmt.Fprintln(out, "源地址切换成功")
			return nil
		}
	}
}

func RunSetCertMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	openSSLDir, err := resolveOpenSSLDir(opts)
	if err != nil {
		return err
	}
	if strings.TrimSpace(openSSLDir) == "" {
		fmt.Fprintln(out, "设备可能尚未安装openssl，无法安装证书文件")
		return nil
	}
	certPath := resolveCertPath(openSSLDir)
	for {
		fmt.Fprintln(out, "安装/更新本地根证书文件（ca-certificates.crt）")
		fmt.Fprintln(out, "用于解决证书校验错误，x509报错等问题")
		if _, err := os.Stat(certPath); err == nil {
			fmt.Fprintf(out, "检测到系统已存在根证书文件：%s\n", certPath)
			fmt.Fprintln(out, "1) 覆盖更新")
		} else {
			fmt.Fprintln(out, "1) 立即安装")
		}
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := installRootCert(opts, openSSLDir, certPath); err != nil {
				return err
			}
			fmt.Fprintln(out, "证书安装成功")
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunSetScriptsMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintln(out, "更新管理脚本")
		fmt.Fprintln(out, "注意：更新时会停止服务")
		fmt.Fprintln(out, "1) 立即更新")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := upgradeRunTask(opts.CrashDir, "update_scripts"); err != nil {
				return err
			}
			fmt.Fprintln(out, "管理脚本更新成功")
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

type geoChoice struct {
	Label      string
	GeoType    string
	GeoName    string
	VersionKey string
}

func RunSetGeoMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	choices := []geoChoice{
		{Label: "CN-IP绕过文件（cn_ip.txt）", GeoType: "china_ip_list.txt", GeoName: "cn_ip.txt", VersionKey: "china_ip_list_v"},
		{Label: "CN-IPV6绕过文件（cn_ipv6.txt）", GeoType: "china_ipv6_list.txt", GeoName: "cn_ipv6.txt", VersionKey: "china_ipv6_list_v"},
		{Label: "Mihomo精简版GeoIP（Country.mmdb）", GeoType: "cn_mini.mmdb", GeoName: "Country.mmdb", VersionKey: "cn_mini_v"},
		{Label: "Mihomo完整版GeoSite（GeoSite.dat）", GeoType: "geosite.dat", GeoName: "GeoSite.dat", VersionKey: "geosite_v"},
		{Label: "Mihomo-mrs常用包（ruleset/*.mrs）", GeoType: "mrs.tar.gz", GeoName: "mrs.tar.gz", VersionKey: "mrs_v"},
		{Label: "Singbox-srs常用包（ruleset/*.srs）", GeoType: "srs.tar.gz", GeoName: "srs.tar.gz", VersionKey: "srs_v"},
	}

	for {
		fmt.Fprintln(out, "安装/更新本地Geo数据库文件")
		for i, c := range choices {
			fmt.Fprintf(out, "%d) %s\n", i+1, c.Label)
		}
		fmt.Fprintln(out, "8) 自定义数据库链接下载")
		fmt.Fprintln(out, "9) 清理数据库文件")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1", "2", "3", "4", "5", "6":
			idx, _ := strconv.Atoi(choice)
			entry := choices[idx-1]
			if err := installGeoAsset(opts, entry.GeoType, entry.GeoName, entry.VersionKey); err != nil {
				return err
			}
			fmt.Fprintf(out, "%s数据库文件下载成功\n", entry.GeoType)
			return nil
		case "8":
			fmt.Fprint(out, "请输入数据库下载链接(0返回)> ")
			url, err := readLine(reader)
			if err != nil {
				return err
			}
			url = strings.TrimSpace(url)
			if url == "" || url == "0" {
				continue
			}
			name := filepath.Base(url)
			if name == "." || name == "/" || name == "" {
				fmt.Fprintln(out, "输入错误")
				continue
			}
			if err := installGeoFromURL(opts, url, name, ""); err != nil {
				return err
			}
			fmt.Fprintf(out, "%s数据库文件下载成功\n", name)
			return nil
		case "9":
			fmt.Fprintln(out, "是否确认清理所有数据库文件？")
			fmt.Fprintln(out, "1) 确认清理")
			fmt.Fprintln(out, "0) 返回")
			fmt.Fprint(out, "请输入对应标号> ")
			confirm, err := readLine(reader)
			if err != nil {
				return err
			}
			if confirm != "1" {
				continue
			}
			if err := clearGeoAssets(opts); err != nil {
				return err
			}
			fmt.Fprintln(out, "所有数据库文件均已清理")
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunSetDBMenu(opts Options, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintln(out, "安装 dashboard 管理面板到本地")
		fmt.Fprintln(out, "1) 安装zashboard面板")
		fmt.Fprintln(out, "2) 安装MetaXD面板")
		fmt.Fprintln(out, "3) 安装Yacd-Meta魔改面板")
		fmt.Fprintln(out, "4) 安装基础面板")
		fmt.Fprintln(out, "5) 安装Meta基础面板")
		fmt.Fprintln(out, "6) 安装Yacd面板")
		fmt.Fprintln(out, "9) 卸载本地面板")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1", "2", "3", "4", "5", "6":
			dbType := map[string]string{
				"1": "zashboard",
				"2": "meta_xd",
				"3": "meta_yacd",
				"4": "clashdb",
				"5": "meta_db",
				"6": "yacd",
			}[choice]
			if err := setDashboardExternalURL(opts.CrashDir, dbType); err != nil {
				return err
			}
			targetDir, hostdir, err := pickDashboardInstallDir(opts, reader, out)
			if err != nil {
				return err
			}
			if targetDir == "" {
				continue
			}
			if err := installDashboard(opts, dbType, targetDir, hostdir); err != nil {
				return err
			}
			fmt.Fprintln(out, "面板安装成功")
			return nil
		case "9":
			fmt.Fprintln(out, "是否卸载本地面板？")
			fmt.Fprintln(out, "1) 确认卸载")
			fmt.Fprintln(out, "0) 返回")
			fmt.Fprint(out, "请输入对应标号> ")
			confirm, err := readLine(reader)
			if err != nil {
				return err
			}
			if confirm != "1" {
				continue
			}
			if err := uninstallDashboard(opts); err != nil {
				return err
			}
			fmt.Fprintln(out, "面板已经卸载")
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func withDefaults(opts Options) Options {
	if strings.TrimSpace(opts.CrashDir) == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	if strings.TrimSpace(opts.CertSourcePath) == "" {
		opts.CertSourcePath = filepath.Join(opts.CrashDir, "bin", "fix", "ca-certificates.crt")
	}
	if strings.TrimSpace(opts.BinDir) == "" {
		opts.BinDir = strings.TrimSpace(os.Getenv("BINDIR"))
		if opts.BinDir == "" {
			opts.BinDir = opts.CrashDir
		}
	}
	return opts
}

func resolveOpenSSLDir(opts Options) (string, error) {
	if strings.TrimSpace(opts.OpenSSLDir) != "" {
		return opts.OpenSSLDir, nil
	}
	out, err := upgradeExecOutput("openssl", "version", "-d")
	if err != nil {
		return "", nil
	}
	if i := strings.IndexByte(out, '"'); i >= 0 {
		j := strings.LastIndexByte(out, '"')
		if j > i {
			return out[i+1 : j], nil
		}
	}
	return "", nil
}

func resolveCertPath(openSSLDir string) string {
	certsDir := filepath.Join(openSSLDir, "certs")
	if st, err := os.Stat(certsDir); err == nil && st.IsDir() {
		return filepath.Join(certsDir, "ca-certificates.crt")
	}
	return "/etc/ssl/certs/ca-certificates.crt"
}

func installRootCert(opts Options, openSSLDir, certPath string) error {
	b, err := os.ReadFile(opts.CertSourcePath)
	if err != nil {
		return err
	}
	certsDir := filepath.Join(openSSLDir, "certs")
	if st, err := os.Stat(certsDir); err == nil && !st.IsDir() {
		if err := os.Remove(certsDir); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(certPath, b, 0o644)
}

func loadState(opts Options) (State, error) {
	cfgPath := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	kv, err := parseKVFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return State{}, err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	return State{
		ReleaseType: stripQuotes(kv["release_type"]),
		URLID:       stripQuotes(kv["url_id"]),
		UpdateURL:   stripQuotes(kv["update_url"]),
	}, nil
}

func loadCoreState(crashDir string) (CoreState, error) {
	kv, err := parseKVFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err != nil && !os.IsNotExist(err) {
		return CoreState{}, err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	st := CoreState{
		CrashCore: strings.TrimSpace(stripQuotes(kv["crashcore"])),
		CPUCore:   strings.TrimSpace(stripQuotes(kv["cpucore"])),
		ZipType:   strings.TrimSpace(stripQuotes(kv["zip_type"])),
		CoreV:     strings.TrimSpace(stripQuotes(kv["core_v"])),
	}
	if st.CrashCore == "" {
		st.CrashCore = "meta"
	}
	if st.CPUCore == "" {
		st.CPUCore = detectCPUCore()
	}
	if st.ZipType == "" {
		st.ZipType = "tar.gz"
	}
	return st, nil
}

func saveState(opts Options, st State) error {
	cfgPath := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	kv, err := parseKVFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	if strings.TrimSpace(st.URLID) == "" {
		delete(kv, "url_id")
	} else {
		kv["url_id"] = st.URLID
	}
	if strings.TrimSpace(st.ReleaseType) == "" {
		delete(kv, "release_type")
	} else {
		kv["release_type"] = st.ReleaseType
	}
	kv["update_url"] = quoteSingle(st.UpdateURL)
	return writeKVFile(cfgPath, kv)
}

func listServerSources(opts Options) ([]ServerEntry, error) {
	path := filepath.Join(opts.CrashDir, "configs", "servers.list")
	entries, err := parseServersList(path)
	if err != nil && os.IsNotExist(err) {
		fallback := filepath.Join(opts.CrashDir, "public", "servers.list")
		return parseServersList(fallback)
	}
	return entries, err
}

func parseServersList(path string) ([]ServerEntry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]ServerEntry, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		if id < 100 || id > 199 {
			continue
		}
		out = append(out, ServerEntry{ID: id, Name: fields[1], URL: fields[2], Raw: line})
	}
	return out, nil
}

func currentURLName(st State, entries []ServerEntry) string {
	if strings.TrimSpace(st.URLID) == "" {
		if strings.TrimSpace(st.UpdateURL) == "" {
			return "未指定"
		}
		return st.UpdateURL
	}
	for _, e := range entries {
		if strconv.Itoa(e.ID) == st.URLID {
			return e.Name
		}
	}
	if strings.TrimSpace(st.UpdateURL) != "" {
		return st.UpdateURL
	}
	return st.URLID
}

func releaseDisplayName(releaseType string) string {
	switch strings.TrimSpace(releaseType) {
	case "":
		return "未指定"
	case "stable":
		return "稳定版"
	case "master":
		return "公测版"
	case "dev":
		return "开发版"
	default:
		return releaseType + "(回退)"
	}
}

func fetchRollbackTags() ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/juewuy/ShellCrash/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := upgradeHTTPDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var payload []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload))
	for _, item := range payload {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func parseKVFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	kv := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return kv, s.Err()
}

func writeKVFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func installCoreAsset(opts Options, crashCore, cpuCore, zipType string) error {
	crashCore = strings.TrimSpace(crashCore)
	if crashCore == "" {
		crashCore = "meta"
	}
	cpuCore = strings.TrimSpace(cpuCore)
	if cpuCore == "" {
		cpuCore = detectCPUCore()
	}
	zipType = strings.TrimSpace(zipType)
	if zipType == "" {
		zipType = "tar.gz"
	}

	target := "clash"
	if strings.Contains(crashCore, "singbox") {
		target = "singbox"
	}
	folders := []string{crashCore}
	if crashCore == "singbox" {
		folders = []string{"singbox", "singboxp"}
	}

	var data []byte
	var lastErr error
	for _, folder := range folders {
		assetPath := filepath.ToSlash(filepath.Join("bin", folder, target+"-linux-"+cpuCore+"."+zipType))
		url, err := resolveAssetURL(opts.CrashDir, assetPath)
		if err != nil {
			lastErr = err
			continue
		}
		data, err = downloadURL(url)
		if err != nil {
			lastErr = err
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return lastErr
	}

	rawCore, err := normalizeCoreBinary(data, zipType)
	if err != nil {
		return err
	}
	binDir := resolveBinDir(opts.CrashDir)
	tmpDir := resolveTmpDir(opts.CrashDir)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	if err := clearOldCoreArchives(binDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(binDir, "CrashCore."+zipType), data, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "CrashCore"), rawCore, 0o755); err != nil {
		return err
	}

	if err := setConfigValue(opts.CrashDir, "crashcore", crashCore); err != nil {
		return err
	}
	if err := setConfigValue(opts.CrashDir, "cpucore", cpuCore); err != nil {
		return err
	}
	if err := setConfigValue(opts.CrashDir, "zip_type", zipType); err != nil {
		return err
	}
	if err := setConfigValue(opts.CrashDir, "custcorelink", ""); err != nil {
		return err
	}
	if strings.Contains(crashCore, "singbox") {
		if err := setEnvValue(opts.CrashDir, "COMMAND", "\"$TMPDIR/CrashCore run -D $BINDIR -C $TMPDIR/jsons\""); err != nil {
			return err
		}
		return nil
	}
	return setEnvValue(opts.CrashDir, "COMMAND", "\"$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml\"")
}

func normalizeCoreBinary(data []byte, zipType string) ([]byte, error) {
	switch zipType {
	case "tar.gz":
		return extractCoreFromArchive(data)
	case "gz":
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		return io.ReadAll(gr)
	case "upx":
		return data, nil
	default:
		return data, nil
	}
}

func extractCoreFromArchive(archive []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	var best []byte
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if h == nil || (h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA) {
			continue
		}
		name := strings.ToLower(filepath.Base(h.Name))
		if !strings.Contains(name, "crashcore") &&
			!strings.Contains(name, "sing") &&
			!strings.Contains(name, "meta") &&
			!strings.Contains(name, "mihomo") &&
			!strings.Contains(name, "clash") &&
			!strings.Contains(name, "pre") {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		if len(b) > len(best) {
			best = b
		}
	}
	if len(best) == 0 {
		return nil, fmt.Errorf("core binary not found in archive")
	}
	return best, nil
}

func resolveBinDir(crashDir string) string {
	if v := strings.TrimSpace(os.Getenv("BINDIR")); v != "" {
		return v
	}
	kv, err := parseEnvFile(filepath.Join(crashDir, "configs", "command.env"))
	if err == nil {
		if v := strings.TrimSpace(kv["BINDIR"]); v != "" {
			return v
		}
	}
	return crashDir
}

func resolveTmpDir(crashDir string) string {
	if v := strings.TrimSpace(os.Getenv("TMPDIR")); v != "" {
		return v
	}
	kv, err := parseKVFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err == nil {
		if v := strings.TrimSpace(stripQuotes(kv["TMPDIR"])); v != "" {
			return v
		}
		if v := strings.TrimSpace(stripQuotes(kv["tmp_dir"])); v != "" {
			return v
		}
	}
	return "/tmp/ShellCrash"
}

func clearOldCoreArchives(binDir string) error {
	for _, name := range []string{"CrashCore.tar.gz", "CrashCore.gz", "CrashCore.upx"} {
		if err := os.Remove(filepath.Join(binDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func runZipTypeMenu(crashDir string, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprintln(out, "请选择内核压缩格式")
	fmt.Fprintln(out, "1) upx")
	fmt.Fprintln(out, "2) tar.gz")
	fmt.Fprintln(out, "3) gz")
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应标号> ")
	choice, err := readLine(reader)
	if err != nil {
		return err
	}
	switch choice {
	case "", "0":
		return nil
	case "1":
		return setConfigValue(crashDir, "zip_type", "upx")
	case "2":
		return setConfigValue(crashDir, "zip_type", "tar.gz")
	case "3":
		return setConfigValue(crashDir, "zip_type", "gz")
	default:
		fmt.Fprintln(out, "输入错误")
		return nil
	}
}

func runCPUCoreMenu(crashDir string, reader *bufio.Reader, out io.Writer) error {
	choices := []string{"armv5", "armv7", "arm64", "386", "amd64", "mipsle-softfloat", "mipsle-hardfloat", "mips-softfloat"}
	fmt.Fprintln(out, "请选择处理器架构")
	for i, c := range choices {
		fmt.Fprintf(out, "%d) %s\n", i+1, c)
	}
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应标号> ")
	choice, err := readLine(reader)
	if err != nil {
		return err
	}
	if choice == "" || choice == "0" {
		return nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(choices) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	return setConfigValue(crashDir, "cpucore", choices[idx-1])
}

func detectCPUCore() string {
	out, err := upgradeExecOutput("uname", "-m")
	if err == nil {
		arch := strings.ToLower(strings.TrimSpace(out))
		switch {
		case strings.Contains(arch, "x86_64"), strings.Contains(arch, "amd64"):
			return "amd64"
		case strings.Contains(arch, "aarch64"), strings.Contains(arch, "arm64"):
			return "arm64"
		case strings.Contains(arch, "armv7"):
			return "armv7"
		case strings.Contains(arch, "armv5"), strings.Contains(arch, "armv6"), strings.Contains(arch, "arm"):
			return "armv5"
		case strings.Contains(arch, "mips64el"), strings.Contains(arch, "mipsel"):
			return "mipsle-softfloat"
		case strings.Contains(arch, "mips"):
			return "mips-softfloat"
		case strings.Contains(arch, "386"), strings.Contains(arch, "i686"), strings.Contains(arch, "i386"):
			return "386"
		}
	}
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "arm":
		return "armv7"
	case "386":
		return "386"
	default:
		return "amd64"
	}
}

func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	kv := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return kv, s.Err()
}

func writeEnvFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func setEnvValue(crashDir, key, value string) error {
	path := filepath.Join(crashDir, "configs", "command.env")
	kv, err := parseEnvFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	if strings.TrimSpace(value) == "" {
		delete(kv, key)
	} else {
		kv[key] = value
	}
	return writeEnvFile(path, kv)
}

func quoteSingle(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "") + "'"
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func installGeoAsset(opts Options, geoType, geoName, versionKey string) error {
	relPath := filepath.ToSlash(filepath.Join("bin", "geodata", geoType))
	url, err := resolveAssetURL(opts.CrashDir, relPath)
	if err != nil {
		return err
	}
	return installGeoFromURL(opts, url, geoName, versionKey)
}

func installGeoFromURL(opts Options, url, geoName, versionKey string) error {
	geoName = strings.TrimSpace(geoName)
	if geoName == "" {
		return fmt.Errorf("invalid geo file name")
	}
	data, err := downloadURL(url)
	if err != nil {
		return err
	}
	if strings.HasSuffix(geoName, ".tar.gz") {
		if err := extractGeoArchive(opts.CrashDir, data); err != nil {
			return err
		}
	} else {
		targetDir := opts.CrashDir
		if strings.HasSuffix(geoName, ".mrs") || strings.HasSuffix(geoName, ".srs") {
			targetDir = filepath.Join(opts.CrashDir, "ruleset")
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(targetDir, geoName), data, 0o644); err != nil {
			return err
		}
	}
	if strings.TrimSpace(versionKey) == "" {
		return nil
	}
	geoVersion, err := fetchGeoVersion(opts.CrashDir)
	if err != nil || strings.TrimSpace(geoVersion) == "" {
		return nil
	}
	return setConfigValue(opts.CrashDir, versionKey, geoVersion)
}

func extractGeoArchive(crashDir string, archive []byte) error {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gr.Close()

	rulesetDir := filepath.Join(crashDir, "ruleset")
	if err := os.MkdirAll(rulesetDir, 0o755); err != nil {
		return err
	}
	root := filepath.Clean(rulesetDir)
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h == nil || (h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA) {
			continue
		}
		name := filepath.Clean(strings.TrimPrefix(h.Name, "/"))
		if strings.HasPrefix(name, "..") {
			return fmt.Errorf("archive entry escapes ruleset dir: %s", h.Name)
		}
		target := filepath.Join(root, name)
		if target != root && !strings.HasPrefix(target, root+string(filepath.Separator)) {
			return fmt.Errorf("archive entry escapes ruleset dir: %s", h.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func clearGeoAssets(opts Options) error {
	for _, name := range []string{
		"cn_ip.txt",
		"cn_ipv6.txt",
		"Country.mmdb",
		"GeoSite.dat",
		"geoip.db",
		"geosite.db",
	} {
		_ = os.Remove(filepath.Join(opts.CrashDir, name))
	}
	rulesetDir := filepath.Join(opts.CrashDir, "ruleset")
	entries, err := os.ReadDir(rulesetDir)
	if err == nil {
		for _, e := range entries {
			_ = os.RemoveAll(filepath.Join(rulesetDir, e.Name()))
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, key := range []string{
		"Country_v", "cn_mini_v", "china_ip_list_v", "china_ipv6_list_v",
		"geosite_v", "geoip_cn_v", "geosite_cn_v", "mrs_geosite_cn_v",
		"srs_geoip_cn_v", "srs_geosite_cn_v", "mrs_v", "srs_v",
	} {
		if err := setConfigValue(opts.CrashDir, key, ""); err != nil {
			return err
		}
	}
	return nil
}

func resolveAssetURL(crashDir, relPath string) (string, error) {
	relPath = strings.TrimLeft(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return "", fmt.Errorf("empty asset path")
	}
	kv, err := parseKVFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if kv == nil {
		kv = map[string]string{}
	}

	updateURL := strings.TrimSpace(stripQuotes(kv["update_url"]))
	if updateURL == "" {
		updateURL = "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master"
	}
	urlID := strings.TrimSpace(stripQuotes(kv["url_id"]))
	if urlID == "" {
		return strings.TrimRight(updateURL, "/") + "/" + relPath, nil
	}

	branch := strings.TrimSpace(stripQuotes(kv["release_type"]))
	if branch == "" {
		branch = "master"
	}
	if strings.HasPrefix(relPath, "bin/") {
		branch = "update"
	} else if strings.HasPrefix(relPath, "public/") || strings.HasPrefix(relPath, "rules/") {
		branch = "dev"
	}
	base := lookupServerBase(crashDir, urlID)
	if base == "" {
		return "", fmt.Errorf("url_id %q not found in servers.list", urlID)
	}
	base = strings.TrimRight(base, "/")
	if urlID == "101" || urlID == "104" {
		return base + "@" + branch + "/" + relPath, nil
	}
	return base + "/" + branch + "/" + relPath, nil
}

func lookupServerBase(crashDir, id string) string {
	paths := []string{
		filepath.Join(crashDir, "configs", "servers.list"),
		filepath.Join(crashDir, "public", "servers.list"),
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == id {
				return fields[2]
			}
		}
	}
	return ""
}

func downloadURL(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := upgradeHTTPDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(resp.Body)
}

func fetchGeoVersion(crashDir string) (string, error) {
	url, err := resolveAssetURL(crashDir, "bin/version")
	if err != nil {
		return "", err
	}
	body, err := downloadURL(url)
	if err != nil {
		return "", err
	}
	for _, raw := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == "GeoIP_v" {
			return stripQuotes(v), nil
		}
	}
	return "", nil
}

func setConfigValue(crashDir, key, value string) error {
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	kv, err := parseKVFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	if strings.TrimSpace(value) == "" {
		delete(kv, key)
	} else {
		kv[key] = value
	}
	return writeKVFile(cfgPath, kv)
}

func setDashboardExternalURL(crashDir, dbType string) error {
	switch dbType {
	case "zashboard":
		return setConfigValue(crashDir, "external_ui_url", "https://github.com/Zephyruso/zashboard/releases/latest/download/dist-cdn-fonts.zip")
	case "meta_xd":
		return setConfigValue(crashDir, "external_ui_url", "https://raw.githubusercontent.com/juewuy/ShellCrash/update/bin/dashboard/meta_xd.tar.gz")
	default:
		return nil
	}
}

func pickDashboardInstallDir(opts Options, reader *bufio.Reader, out io.Writer) (string, string, error) {
	wwwClash := "/www/clash"
	crashUI := filepath.Join(opts.CrashDir, "ui")
	if fileExists(filepath.Join(wwwClash, "CNAME")) || fileExists(filepath.Join(crashUI, "CNAME")) {
		fmt.Fprintln(out, "检测到已经安装过本地面板")
		fmt.Fprintln(out, "1) 升级/覆盖安装")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		res, err := readLine(reader)
		if err != nil {
			return "", "", err
		}
		if res != "1" {
			return "", "", nil
		}
		if fileExists(filepath.Join(wwwClash, "CNAME")) {
			_ = os.RemoveAll(wwwClash)
			_ = os.RemoveAll(filepath.Join(opts.BinDir, "ui"))
			return wwwClash, "/clash", nil
		}
		_ = os.RemoveAll(crashUI)
		_ = os.RemoveAll(filepath.Join(opts.BinDir, "ui"))
		return crashUI, ":" + defaultDBPort(opts.CrashDir) + "/ui", nil
	}

	if canUseWWWInstall() {
		fmt.Fprintln(out, "请选择面板安装目录：")
		fmt.Fprintf(out, "1) 在%s安装\n", crashUI)
		fmt.Fprintln(out, "2) 在/www/clash目录安装")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")
		num, err := readLine(reader)
		if err != nil {
			return "", "", err
		}
		switch num {
		case "", "0":
			return "", "", nil
		case "1":
			return crashUI, ":" + defaultDBPort(opts.CrashDir) + "/ui", nil
		case "2":
			return wwwClash, "/clash", nil
		default:
			fmt.Fprintln(out, "输入错误")
			return "", "", nil
		}
	}
	return crashUI, ":" + defaultDBPort(opts.CrashDir) + "/ui", nil
}

func installDashboard(opts Options, dbType, targetDir, hostdir string) error {
	url, err := resolveAssetURL(opts.CrashDir, filepath.ToSlash(filepath.Join("bin", "dashboard", dbType+".tar.gz")))
	if err != nil {
		return err
	}
	data, err := downloadURL(url)
	if err != nil {
		return err
	}
	_ = os.RemoveAll(targetDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	if err := extractTarGzToDir(targetDir, data); err != nil {
		return err
	}
	host := defaultHost(opts.CrashDir)
	port := defaultDBPort(opts.CrashDir)
	if err := patchDashboardHostPort(targetDir, dbType, host, port); err != nil {
		return err
	}
	if err := setConfigValue(opts.CrashDir, "hostdir", quoteSingle(hostdir)); err != nil {
		return err
	}
	return nil
}

func extractTarGzToDir(targetDir string, archive []byte) error {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gr.Close()

	root := filepath.Clean(targetDir)
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h == nil {
			continue
		}
		name := filepath.Clean(strings.TrimPrefix(h.Name, "/"))
		if strings.HasPrefix(name, "..") {
			return fmt.Errorf("archive entry escapes target dir: %s", h.Name)
		}
		target := filepath.Join(root, name)
		if target != root && !strings.HasPrefix(target, root+string(filepath.Separator)) {
			return fmt.Errorf("archive entry escapes target dir: %s", h.Name)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func patchDashboardHostPort(targetDir, dbType, host, port string) error {
	old := "127.0.0.1:9090"
	newVal := host + ":" + port
	if dbType == "clashdb" || dbType == "meta_db" || dbType == "zashboard" {
		files, err := filepath.Glob(filepath.Join(targetDir, "assets", "*.js"))
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := replaceInFile(f, "127.0.0.1", host); err != nil {
				return err
			}
			if err := replaceInFile(f, "9090", port); err != nil {
				return err
			}
		}
		return nil
	}
	if dbType == "meta_xd" {
		files, err := filepath.Glob(filepath.Join(targetDir, "_nuxt", "*.js"))
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := replaceInFile(f, old, newVal); err != nil {
				return err
			}
		}
		return nil
	}

	files, err := filepath.Glob(filepath.Join(targetDir, "*.html"))
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := replaceInFile(f, old, newVal); err != nil {
			return err
		}
	}
	return nil
}

func replaceInFile(path, old, newVal string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next := strings.ReplaceAll(string(b), old, newVal)
	return os.WriteFile(path, []byte(next), 0o644)
}

func uninstallDashboard(opts Options) error {
	_ = os.RemoveAll("/www/clash")
	_ = os.RemoveAll(filepath.Join(opts.CrashDir, "ui"))
	_ = os.RemoveAll(filepath.Join(opts.BinDir, "ui"))
	return nil
}

func defaultHost(crashDir string) string {
	kv, err := parseKVFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err == nil {
		if v := strings.TrimSpace(stripQuotes(kv["host"])); v != "" {
			return v
		}
	}
	return "127.0.0.1"
}

func defaultDBPort(crashDir string) string {
	kv, err := parseKVFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err == nil {
		if v := strings.TrimSpace(stripQuotes(kv["db_port"])); v != "" {
			return v
		}
	}
	return "9999"
}

func canUseWWWInstall() bool {
	if st, err := os.Stat("/www"); err != nil || !st.IsDir() {
		return false
	}
	testPath := filepath.Join("/www", ".shellcrash-upgradectl-write-test")
	if err := os.WriteFile(testPath, []byte("x"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(testPath)
	out, err := upgradeExecOutput("pidof", "nginx")
	return err == nil && strings.TrimSpace(out) != ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
