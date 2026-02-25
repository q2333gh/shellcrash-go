package upgradectl

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shellcrash/internal/uninstallctl"
)

func TestRunSetServerMenuSelectBuiltInSource(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "servers.list"), []byte("101 MirrorA https://a.example\n102 MirrorB https://b.example\n"), 0o644); err != nil {
		t.Fatalf("write servers.list: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("release_type=stable\nurl_id=101\nupdate_url=''\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("2\n")
	if err := RunSetServerMenu(Options{CrashDir: crashDir}, in, io.Discard); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if got := kv["url_id"]; got != "102" {
		t.Fatalf("url_id mismatch: got=%q", got)
	}
	if got := kv["release_type"]; got != "stable" {
		t.Fatalf("release_type mismatch: got=%q", got)
	}
	if got := kv["update_url"]; got != "''" {
		t.Fatalf("update_url mismatch: got=%q", got)
	}
}

func TestRunSetServerMenuCustomSource(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "servers.list"), []byte("101 MirrorA https://a.example\n"), 0o644); err != nil {
		t.Fatalf("write servers.list: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("release_type=stable\nurl_id=101\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("d\nhttps://custom.example/ShellCrash\n")
	if err := RunSetServerMenu(Options{CrashDir: crashDir}, in, io.Discard); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if _, ok := kv["url_id"]; ok {
		t.Fatalf("url_id should be removed")
	}
	if _, ok := kv["release_type"]; ok {
		t.Fatalf("release_type should be removed")
	}
	if got := kv["update_url"]; got != "'https://custom.example/ShellCrash'" {
		t.Fatalf("update_url mismatch: got=%q", got)
	}
}

func TestRunSetServerMenuRollbackTagSelection(t *testing.T) {
	oldHTTP := upgradeHTTPDo
	defer func() { upgradeHTTPDo = oldHTTP }()
	upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(strings.NewReader(`[ {"name":"1.9.0"}, {"name":"master-2025"}, {"name":"1.8.9"} ]`))
		return &http.Response{StatusCode: 200, Body: body}, nil
	}

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "servers.list"), []byte("101 MirrorA https://a.example\n"), 0o644); err != nil {
		t.Fatalf("write servers.list: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("release_type=stable\nurl_id=101\nupdate_url='https://old.example'\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	in := strings.NewReader("e\n2\n")
	if err := RunSetServerMenu(Options{CrashDir: crashDir}, in, io.Discard); err != nil {
		t.Fatalf("run menu: %v", err)
	}

	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if got := kv["release_type"]; got != "1.8.9" {
		t.Fatalf("release_type mismatch: got=%q", got)
	}
	if got := kv["update_url"]; got != "''" {
		t.Fatalf("update_url mismatch: got=%q", got)
	}
}

func TestRunSetCertMenuInstallWritesTargetCert(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	sourcePath := filepath.Join(crashDir, "bin", "fix", "ca-certificates.crt")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("CERTDATA"), 0o644); err != nil {
		t.Fatalf("write source cert: %v", err)
	}

	openSSLDir := filepath.Join(td, "openssl")
	if err := os.MkdirAll(filepath.Join(openSSLDir, "certs"), 0o755); err != nil {
		t.Fatalf("mkdir openssl certs dir: %v", err)
	}

	in := strings.NewReader("1\n")
	var out strings.Builder
	if err := RunSetCertMenu(Options{
		CrashDir:   crashDir,
		OpenSSLDir: openSSLDir,
	}, in, &out); err != nil {
		t.Fatalf("run setcrt menu: %v", err)
	}

	target := filepath.Join(openSSLDir, "certs", "ca-certificates.crt")
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target cert: %v", err)
	}
	if string(got) != "CERTDATA" {
		t.Fatalf("unexpected cert content: %q", string(got))
	}
}

func TestRunSetCertMenuWithoutOpenSSLShowsHint(t *testing.T) {
	oldExec := upgradeExecOutput
	defer func() { upgradeExecOutput = oldExec }()
	upgradeExecOutput = func(name string, args ...string) (string, error) {
		return "", os.ErrNotExist
	}

	in := strings.NewReader("1\n")
	var out strings.Builder
	if err := RunSetCertMenu(Options{CrashDir: t.TempDir()}, in, &out); err != nil {
		t.Fatalf("run setcrt menu: %v", err)
	}
	if !strings.Contains(out.String(), "尚未安装openssl") {
		t.Fatalf("expected openssl hint, got=%q", out.String())
	}
}

func TestRunSetScriptsMenuDispatchesUpdateScriptsTask(t *testing.T) {
	oldRunTask := upgradeRunTask
	defer func() { upgradeRunTask = oldRunTask }()

	called := make([]string, 0, 1)
	upgradeRunTask = func(crashDir, action string) error {
		called = append(called, crashDir+"|"+action)
		return nil
	}

	var out strings.Builder
	if err := RunSetScriptsMenu(Options{CrashDir: "/x/crash"}, strings.NewReader("1\n"), &out); err != nil {
		t.Fatalf("run setscripts menu: %v", err)
	}
	if len(called) != 1 || called[0] != "/x/crash|update_scripts" {
		t.Fatalf("unexpected dispatch: %#v", called)
	}
	if !strings.Contains(out.String(), "更新成功") {
		t.Fatalf("expected success message, got=%q", out.String())
	}
}

func TestRunSetScriptsMenuReturnWithoutDispatch(t *testing.T) {
	oldRunTask := upgradeRunTask
	defer func() { upgradeRunTask = oldRunTask }()

	called := false
	upgradeRunTask = func(crashDir, action string) error {
		called = true
		return nil
	}

	if err := RunSetScriptsMenu(Options{CrashDir: "/x/crash"}, strings.NewReader("0\n"), io.Discard); err != nil {
		t.Fatalf("run setscripts menu: %v", err)
	}
	if called {
		t.Fatal("expected no task dispatch on return")
	}
}

func TestRunSetGeoMenuDownloadsGeoFileAndWritesVersion(t *testing.T) {
	oldHTTP := upgradeHTTPDo
	defer func() { upgradeHTTPDo = oldHTTP }()
	upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
		body := "DATA"
		if strings.HasSuffix(req.URL.Path, "/bin/version") {
			body = "GeoIP_v=2026.02.25\n"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("update_url='https://mirror.example/ShellCrash'\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	if err := RunSetGeoMenu(Options{CrashDir: crashDir}, strings.NewReader("1\n"), io.Discard); err != nil {
		t.Fatalf("run setgeo: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(crashDir, "cn_ip.txt"))
	if err != nil {
		t.Fatalf("read geo file: %v", err)
	}
	if string(data) != "DATA" {
		t.Fatalf("unexpected geo content: %q", string(data))
	}
	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if got := kv["china_ip_list_v"]; got != "2026.02.25" {
		t.Fatalf("unexpected china_ip_list_v: %q", got)
	}
}

func TestRunSetGeoMenuExtractsRulesetArchive(t *testing.T) {
	oldHTTP := upgradeHTTPDo
	defer func() { upgradeHTTPDo = oldHTTP }()
	archive := buildTarGz(t, map[string]string{
		"geo/test.mrs": "MRS",
	})
	upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
		var body io.Reader = bytes.NewReader(archive)
		if strings.HasSuffix(req.URL.Path, "/bin/version") {
			body = strings.NewReader("GeoIP_v=2026.02.25\n")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(body)}, nil
	}

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("update_url='https://mirror.example/ShellCrash'\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	if err := RunSetGeoMenu(Options{CrashDir: crashDir}, strings.NewReader("5\n"), io.Discard); err != nil {
		t.Fatalf("run setgeo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(crashDir, "ruleset", "geo", "test.mrs")); err != nil {
		t.Fatalf("expected extracted ruleset file: %v", err)
	}
}

func TestRunSetGeoMenuCleanupRemovesFilesAndVersionKeys(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	rulesetDir := filepath.Join(crashDir, "ruleset")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.MkdirAll(rulesetDir, 0o755); err != nil {
		t.Fatalf("mkdir ruleset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("cn_mini_v=1\nmrs_v=2\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	for _, f := range []string{"cn_ip.txt", "Country.mmdb"} {
		if err := os.WriteFile(filepath.Join(crashDir, f), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	if err := os.WriteFile(filepath.Join(rulesetDir, "x.mrs"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write ruleset: %v", err)
	}

	if err := RunSetGeoMenu(Options{CrashDir: crashDir}, strings.NewReader("9\n1\n"), io.Discard); err != nil {
		t.Fatalf("run setgeo cleanup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(crashDir, "cn_ip.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected cn_ip.txt removed, err=%v", err)
	}
	entries, err := os.ReadDir(rulesetDir)
	if err != nil {
		t.Fatalf("read ruleset dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected ruleset dir empty, got=%d", len(entries))
	}
	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if _, ok := kv["cn_mini_v"]; ok {
		t.Fatalf("expected cn_mini_v removed")
	}
	if _, ok := kv["mrs_v"]; ok {
		t.Fatalf("expected mrs_v removed")
	}
}

func TestRunSetDBMenuInstallsDashboardAndPatchesHostPort(t *testing.T) {
	oldHTTP := upgradeHTTPDo
	defer func() { upgradeHTTPDo = oldHTTP }()
	archive := buildTarGz(t, map[string]string{
		"assets/app.js": "fetch('http://127.0.0.1:9090')",
		"CNAME":         "ui.example",
	})
	upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(archive))}, nil
	}

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	cfg := "update_url='https://mirror.example/ShellCrash'\ndb_port=10090\nhost='192.168.50.1'\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	if err := RunSetDBMenu(Options{CrashDir: crashDir}, strings.NewReader("4\n"), io.Discard); err != nil {
		t.Fatalf("run setdb: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(crashDir, "ui", "assets", "app.js"))
	if err != nil {
		t.Fatalf("read dashboard js: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "192.168.50.1") || !strings.Contains(text, "10090") {
		t.Fatalf("expected host/port rewrite, got: %s", text)
	}
	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if got := kv["hostdir"]; got != "':10090/ui'" {
		t.Fatalf("unexpected hostdir: %q", got)
	}
}

func TestRunSetDBMenuSetsExternalURLForZashboard(t *testing.T) {
	oldHTTP := upgradeHTTPDo
	defer func() { upgradeHTTPDo = oldHTTP }()
	archive := buildTarGz(t, map[string]string{"assets/app.js": "http://127.0.0.1:9090"})
	upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(archive))}, nil
	}

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte("update_url='https://mirror.example/ShellCrash'\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	if err := RunSetDBMenu(Options{CrashDir: crashDir}, strings.NewReader("1\n"), io.Discard); err != nil {
		t.Fatalf("run setdb: %v", err)
	}
	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	want := "https://github.com/Zephyruso/zashboard/releases/latest/download/dist-cdn-fonts.zip"
	if got := kv["external_ui_url"]; got != want {
		t.Fatalf("unexpected external_ui_url: got=%q want=%q", got, want)
	}
}

func TestRunSetDBMenuUninstallsDashboardPaths(t *testing.T) {
	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	binDir := filepath.Join(td, "bin")
	if err := os.MkdirAll(filepath.Join(crashDir, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir crash ui: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(binDir, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir bin ui: %v", err)
	}

	if err := RunSetDBMenu(Options{CrashDir: crashDir, BinDir: binDir}, strings.NewReader("9\n1\n"), io.Discard); err != nil {
		t.Fatalf("run setdb uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(crashDir, "ui")); !os.IsNotExist(err) {
		t.Fatalf("expected crash ui removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "ui")); !os.IsNotExist(err) {
		t.Fatalf("expected bin ui removed, err=%v", err)
	}
}

func TestRunUpgradeMenuDispatchesSetCore(t *testing.T) {
	oldRunSetCore := upgradeRunSetCoreMenu
	oldRunSetScripts := upgradeRunSetScriptsMenu
	oldRunSetGeo := upgradeRunSetGeoMenu
	oldRunSetDB := upgradeRunSetDBMenu
	oldRunSetCert := upgradeRunSetCertMenu
	oldRunSetServer := upgradeRunSetServerMenu
	oldRunUninstall := upgradeRunUninstallMenu
	defer func() {
		upgradeRunSetCoreMenu = oldRunSetCore
		upgradeRunSetScriptsMenu = oldRunSetScripts
		upgradeRunSetGeoMenu = oldRunSetGeo
		upgradeRunSetDBMenu = oldRunSetDB
		upgradeRunSetCertMenu = oldRunSetCert
		upgradeRunSetServerMenu = oldRunSetServer
		upgradeRunUninstallMenu = oldRunUninstall
	}()

	called := ""
	upgradeRunSetCoreMenu = func(opts Options, in io.Reader, out io.Writer) error {
		called = "setcore"
		return nil
	}
	upgradeRunSetScriptsMenu = func(opts Options, in io.Reader, out io.Writer) error { return nil }
	upgradeRunSetGeoMenu = func(opts Options, in io.Reader, out io.Writer) error { return nil }
	upgradeRunSetDBMenu = func(opts Options, in io.Reader, out io.Writer) error { return nil }
	upgradeRunSetCertMenu = func(opts Options, in io.Reader, out io.Writer) error { return nil }
	upgradeRunSetServerMenu = func(opts Options, in io.Reader, out io.Writer) error { return nil }
	upgradeRunUninstallMenu = func(opts uninstallctl.Options, in io.Reader, out io.Writer) error { return nil }

	if err := RunUpgradeMenu(Options{CrashDir: "/tmp/x"}, strings.NewReader("2\n"), io.Discard); err != nil {
		t.Fatalf("run upgrade menu: %v", err)
	}
	if called != "setcore" {
		t.Fatalf("unexpected dispatch: %q", called)
	}
}

func TestRunSetCoreMenuInstallsCoreArchiveAndUpdatesConfig(t *testing.T) {
	oldHTTP := upgradeHTTPDo
	defer func() { upgradeHTTPDo = oldHTTP }()
	archive := buildTarGz(t, map[string]string{
		"core/mihomo": "COREBIN",
	})
	upgradeHTTPDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(archive))}, nil
	}

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	cfgDir := filepath.Join(crashDir, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	cfg := "update_url='https://mirror.example/ShellCrash'\ncpucore=amd64\nzip_type=tar.gz\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "command.env"), []byte("BINDIR="+crashDir+"\n"), 0o644); err != nil {
		t.Fatalf("write command.env: %v", err)
	}

	if err := RunSetCoreMenu(Options{CrashDir: crashDir}, strings.NewReader("1\n"), io.Discard); err != nil {
		t.Fatalf("run setcore: %v", err)
	}

	if _, err := os.Stat(filepath.Join(crashDir, "CrashCore.tar.gz")); err != nil {
		t.Fatalf("expected core archive installed: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join("/tmp/ShellCrash", "CrashCore"))
	if err != nil {
		t.Fatalf("read extracted core: %v", err)
	}
	if string(raw) != "COREBIN" {
		t.Fatalf("unexpected extracted core: %q", string(raw))
	}
	kv, err := parseKVFile(filepath.Join(cfgDir, "ShellCrash.cfg"))
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if got := kv["crashcore"]; got != "meta" {
		t.Fatalf("unexpected crashcore: %q", got)
	}
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}
