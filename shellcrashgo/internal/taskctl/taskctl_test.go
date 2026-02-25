package taskctl

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"shellcrash/internal/initctl"
	"shellcrash/internal/coreconfig"
	"shellcrash/internal/startctl"
)

func TestResolveTaskCommand(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := "101#echo one#任务一\n102#echo two#任务二\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, name, err := resolveTaskCommand(root, "102")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "echo two" || name != "任务二" {
		t.Fatalf("unexpected task values: cmd=%q name=%q", cmd, name)
	}
}

func TestRunUpdateConfigCallsGoPipeline(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	env := "TMPDIR='" + filepath.Join(root, "tmp") + "'\nBINDIR='" + root + "'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	origCore := coreConfigRun
	origStart := startActionRun
	defer func() {
		coreConfigRun = origCore
		startActionRun = origStart
	}()

	coreCalled := false
	startCalled := false
	coreConfigRun = func(opts coreconfig.Options) (coreconfig.Result, error) {
		coreCalled = true
		if opts.CrashDir != root {
			t.Fatalf("unexpected crash dir: %s", opts.CrashDir)
		}
		return coreconfig.Result{}, nil
	}
	startActionRun = func(crashDir, action string) error {
		startCalled = true
		if crashDir != root || action != "start" {
			t.Fatalf("unexpected start call: %s %s", crashDir, action)
		}
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"update_config"}); err != nil {
		t.Fatal(err)
	}
	if !coreCalled || !startCalled {
		t.Fatalf("expected update_config pipeline to call core/start, core=%v start=%v", coreCalled, startCalled)
	}
}

func TestRunTaskIDStartScriptUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "101#$CRASHDIR/start.sh restart#重启服务\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := "TMPDIR='" + filepath.Join(root, "tmp") + "'\nBINDIR='" + root + "'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	origStart := startActionRun
	origStartWithArgs := startActionRunWithArgs
	defer func() {
		startActionRun = origStart
		startActionRunWithArgs = origStartWithArgs
	}()

	called := false
	startActionRunWithArgs = func(crashDir, action string, extraArgs []string) error {
		called = true
		if crashDir != root || action != "restart" {
			t.Fatalf("unexpected start action: %s %s", crashDir, action)
		}
		if len(extraArgs) != 0 {
			t.Fatalf("unexpected action args: %v", extraArgs)
		}
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"101"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected start action to use Go path")
	}
}

func TestRunTaskIDStartScriptWithArgsUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "101#$CRASHDIR/start.sh cronset ShellCrash初始化 2 0 * * * echo init #任务\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := "TMPDIR='" + filepath.Join(root, "tmp") + "'\nBINDIR='" + root + "'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	origStart := startActionRun
	origStartWithArgs := startActionRunWithArgs
	defer func() {
		startActionRun = origStart
		startActionRunWithArgs = origStartWithArgs
	}()

	var gotAction string
	var gotArgs []string
	startActionRunWithArgs = func(crashDir, action string, extraArgs []string) error {
		if crashDir != root {
			t.Fatalf("unexpected crashDir: %s", crashDir)
		}
		gotAction = action
		gotArgs = append([]string{}, extraArgs...)
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"101"}); err != nil {
		t.Fatal(err)
	}
	if gotAction != "cronset" {
		t.Fatalf("unexpected action: %s", gotAction)
	}
	wantArgs := []string{"ShellCrash初始化", "2", "0", "*", "*", "*", "echo", "init"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected action args: got=%v want=%v", gotArgs, wantArgs)
	}
}

func TestRunTaskIDTaskScriptUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "106#$CRASHDIR/task/task.sh web_save_auto#自动保存面板配置\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\ndb_port=9090\nsecret='abc'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	env := "TMPDIR='" + filepath.Join(root, "tmp") + "'\nBINDIR='" + root + "'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	origWebSave := webSaveRun
	defer func() { webSaveRun = origWebSave }()

	called := false
	webSaveRun = func(crashDir, tmpDir, dbPort, secret string) error {
		called = true
		if crashDir != root {
			t.Fatalf("unexpected crashDir: %s", crashDir)
		}
		if dbPort != "9090" || secret != "abc" {
			t.Fatalf("unexpected web save args: dbPort=%s secret=%s", dbPort, secret)
		}
		if tmpDir != filepath.Join(root, "tmp") {
			t.Fatalf("unexpected tmpDir: %s", tmpDir)
		}
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"106"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected task.sh action to use Go builtin handler")
	}
}

func TestRunTaskIDTaskScriptUnknownActionFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "106#$CRASHDIR/task/task.sh unknown_action#未知动作\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	env := "TMPDIR='" + filepath.Join(root, "tmp") + "'\nBINDIR='" + root + "'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Runner{CrashDir: root}
	err := r.Run([]string{"106"})
	if err == nil || !strings.Contains(err.Error(), `unsupported task action "unknown_action"`) {
		t.Fatalf("expected unsupported task action error, got: %v", err)
	}
}

func TestRunTaskIDDirectCommandPipelineUsesExecWithoutShell(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "122#sleep 1 && touch $TMPDIR/reboot.flag && reboot#重启路由设备-慎用\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origRunTask := runTaskCommand
	defer func() { runTaskCommand = origRunTask }()

	var got []string
	runTaskCommand = func(cfg startctl.Config, name string, args ...string) error {
		got = append(got, strings.Join(append([]string{name}, args...), " "))
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"122"}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"sleep 1",
		"touch " + filepath.Join(tmpDir, "reboot.flag"),
		"reboot",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected direct pipeline commands: got=%v want=%v", got, want)
	}
}

func TestRunTaskIDDirectCommandPipelineWithQuotesUsesExecWithoutShell(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "122#echo \"a b\" && touch \"$TMPDIR/reboot flag\"#quoted pipeline\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origRunTask := runTaskCommand
	defer func() { runTaskCommand = origRunTask }()

	var got []string
	runTaskCommand = func(cfg startctl.Config, name string, args ...string) error {
		got = append(got, strings.Join(append([]string{name}, args...), " | "))
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"122"}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"echo | a b",
		"touch | " + filepath.Join(tmpDir, "reboot flag"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected quoted pipeline commands: got=%v want=%v", got, want)
	}
}

func TestRunTaskIDShellExpressionNowErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	taskList := "122#echo ok ; reboot#unsupported shell expression\n"
	if err := os.WriteFile(filepath.Join(root, "task", "task.list"), []byte(taskList), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+filepath.Join(root, "tmp")+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Runner{CrashDir: root}
	err := r.Run([]string{"122"})
	if err == nil || !strings.Contains(err.Error(), "unsupported shell expression in task command") {
		t.Fatalf("expected unsupported shell expression error, got: %v", err)
	}
}

func TestRunUnknownTaskCommandArgsExecutesWithoutShell(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+filepath.Join(root, "tmp")+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origRunTask := runTaskCommand
	defer func() { runTaskCommand = origRunTask }()

	var gotName string
	var gotArgs []string
	runTaskCommand = func(cfg startctl.Config, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string{}, args...)
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"echo", "hello"}); err != nil {
		t.Fatal(err)
	}
	if gotName != "echo" || !reflect.DeepEqual(gotArgs, []string{"hello"}) {
		t.Fatalf("unexpected command execution: name=%s args=%v", gotName, gotArgs)
	}
}

func TestWriteTaskLogTruncates(t *testing.T) {
	tmp := t.TempDir()
	for i := 0; i < 210; i++ {
		if err := writeTaskLog(tmp, "line"); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(tmp, "ShellCrash.log"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range splitNonEmpty(string(b)) {
		if line != "" {
			count++
		}
	}
	if count > 199 {
		t.Fatalf("expected <=199 log lines, got %d", count)
	}
}

func TestRunWebSaveAutoUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "crashcore=meta\ndb_port=9999\nsecret='token'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	env := "TMPDIR='" + filepath.Join(root, "tmp") + "'\nBINDIR='" + root + "'\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := webSaveRun
	defer func() { webSaveRun = orig }()

	called := false
	webSaveRun = func(crashDir, tmpDir, dbPort, secret string) error {
		called = true
		if crashDir != root {
			t.Fatalf("unexpected crashDir: %s", crashDir)
		}
		if dbPort != "9999" || secret != "token" {
			t.Fatalf("unexpected auth/port: db=%s secret=%s", dbPort, secret)
		}
		if tmpDir != filepath.Join(root, "tmp") {
			t.Fatalf("unexpected tmpDir: %s", tmpDir)
		}
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"web_save_auto"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected Go web_save handler to be called")
	}
}

func TestRunResetFirewallUsesAfstart(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origStart := startActionRun
	defer func() { startActionRun = origStart }()

	var calls []string
	startActionRun = func(crashDir, action string) error {
		if crashDir != root {
			t.Fatalf("unexpected crashDir: %s", crashDir)
		}
		calls = append(calls, action)
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"reset_firewall"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"stop_firewall", "afstart"}) {
		t.Fatalf("unexpected reset_firewall actions: %v", calls)
	}
}

func TestRunHotUpdateUsesGoPipeline(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("crashcore=meta\ndb_port=8899\nsecret='abc'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "CrashCore"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	origCore := coreConfigRun
	origHTTP := taskHTTPDo
	defer func() {
		coreConfigRun = origCore
		taskHTTPDo = origHTTP
	}()

	coreCalled := false
	coreConfigRun = func(opts coreconfig.Options) (coreconfig.Result, error) {
		coreCalled = true
		return coreconfig.Result{Format: "yaml", CoreConfig: filepath.Join(root, "yamls", "config.yaml")}, nil
	}
	httpCalled := false
	taskHTTPDo = func(req *http.Request) (*http.Response, error) {
		httpCalled = true
		if req.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.String() != "http://127.0.0.1:8899/configs" {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		if got := req.Header.Get("Authorization"); got != "Bearer abc" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		b, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if !strings.Contains(string(b), `"path":"`+filepath.Join(root, "yamls", "config.yaml")+`"`) {
			t.Fatalf("unexpected body: %s", string(b))
		}
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"hotupdate"}); err != nil {
		t.Fatal(err)
	}
	if !coreCalled || !httpCalled {
		t.Fatalf("expected hotupdate pipeline called, core=%v http=%v", coreCalled, httpCalled)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "CrashCore")); !os.IsNotExist(err) {
		t.Fatalf("expected tmp CrashCore removed, err=%v", err)
	}
}

func TestRunUpdateMMDBUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public", "servers.list"), []byte("104 test https://example.com/gh/juewuy/ShellCrash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgText := "crashcore=meta\nurl_id=104\ncn_mini_v=20240101\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfgText), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Country.mmdb"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	origHTTP := taskHTTPDo
	defer func() { taskHTTPDo = origHTTP }()

	var urls []string
	taskHTTPDo = func(req *http.Request) (*http.Response, error) {
		urls = append(urls, req.URL.String())
		switch req.URL.String() {
		case "https://example.com/gh/juewuy/ShellCrash@update/bin/version":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("GeoIP_v=20251205\n")),
			}, nil
		case "https://example.com/gh/juewuy/ShellCrash@update/bin/geodata/cn_mini.mmdb":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("new-mmdb")),
			}, nil
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"update_mmdb"}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(root, "Country.mmdb"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-mmdb" {
		t.Fatalf("unexpected mmdb content: %s", string(got))
	}
	cfgOut, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgOut), "cn_mini_v=20251205") {
		t.Fatalf("expected cn_mini_v updated, got: %s", string(cfgOut))
	}
	if len(urls) < 2 {
		t.Fatalf("expected version + geodata downloads, got: %v", urls)
	}
}

func TestRunUpdateCoreUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public", "servers.list"), []byte("104 test https://example.com/gh/juewuy/ShellCrash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgText := "crashcore=meta\ncore_v=v1.0.0\nurl_id=104\ncpucore=amd64\nzip_type=upx\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfgText), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origHTTP := taskHTTPDo
	origStart := startActionRun
	defer func() {
		taskHTTPDo = origHTTP
		startActionRun = origStart
	}()

	startCalled := false
	startActionRun = func(crashDir, action string) error {
		startCalled = true
		if crashDir != root || action != "start" {
			t.Fatalf("unexpected start action: %s %s", crashDir, action)
		}
		return nil
	}

	coreScript := "#!/bin/sh\nif [ \"$1\" = \"-h\" ]; then echo 'Usage: CrashCore -t'; exit 0; fi\nif [ \"$1\" = \"-v\" ]; then echo 'Mihomo Meta v9.9.9 linux amd64'; exit 0; fi\nexit 0\n"
	taskHTTPDo = func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://example.com/gh/juewuy/ShellCrash@update/bin/version":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("meta_v=v9.9.9\n")),
			}, nil
		case "https://example.com/gh/juewuy/ShellCrash@update/bin/meta/clash-linux-amd64.upx":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(coreScript)),
			}, nil
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"update_core"}); err != nil {
		t.Fatal(err)
	}
	if !startCalled {
		t.Fatal("expected start action after update_core")
	}

	cfgOut, err := os.ReadFile(filepath.Join(root, "configs", "ShellCrash.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgOut), "core_v=v9.9.9") {
		t.Fatalf("expected core_v updated, got: %s", string(cfgOut))
	}

	envOut, err := os.ReadFile(filepath.Join(root, "configs", "command.env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envOut), `COMMAND="$TMPDIR/CrashCore -d $BINDIR -f $TMPDIR/config.yaml"`) {
		t.Fatalf("expected COMMAND updated, got: %s", string(envOut))
	}

	b, err := os.ReadFile(filepath.Join(tmpDir, "CrashCore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Mihomo Meta") {
		t.Fatalf("unexpected core content: %s", string(b))
	}
}

func TestRunUpdateScriptsUsesGoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public", "servers.list"), []byte("104 test https://example.com/gh/juewuy/ShellCrash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "version"), []byte("1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgText := "crashcore=meta\nurl_id=104\n"
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte(cfgText), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	archiveBytes := makeTarGz(t, map[string]string{
		"version":                     "1.1.0\n",
		"configs/new_from_update.txt": "ok\n",
	})

	origHTTP := taskHTTPDo
	origStart := startActionRun
	origInit := initctlRun
	defer func() {
		taskHTTPDo = origHTTP
		startActionRun = origStart
		initctlRun = origInit
	}()

	var actions []string
	startActionRun = func(crashDir, action string) error {
		if crashDir != root {
			t.Fatalf("unexpected crashdir: %s", crashDir)
		}
		actions = append(actions, action)
		return nil
	}
	initCalled := false
	initctlRun = func(opts initctl.Options) error {
		initCalled = true
		if opts.CrashDir != root || opts.TmpDir != tmpDir {
			t.Fatalf("unexpected init options: %+v", opts)
		}
		return nil
	}
	taskHTTPDo = func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://example.com/gh/juewuy/ShellCrash@master/version":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("versionsh=1.1.0\n"))}, nil
		case "https://example.com/gh/juewuy/ShellCrash@master/ShellCrash.tar.gz":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(archiveBytes))}, nil
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"update_scripts"}); err != nil {
		t.Fatal(err)
	}
	if !initCalled {
		t.Fatal("expected initctl to run")
	}
	if !reflect.DeepEqual(actions, []string{"stop", "start"}) {
		t.Fatalf("unexpected start actions: %v", actions)
	}
	if !fileExists(filepath.Join(root, "configs", "new_from_update.txt")) {
		t.Fatal("expected archive extracted into crashdir")
	}
}

func TestRunUpdateScriptsSkipsWhenNoNewVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public", "servers.list"), []byte("104 test https://example.com/gh/juewuy/ShellCrash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "version"), []byte("1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "ShellCrash.cfg"), []byte("url_id=104\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.WriteFile(filepath.Join(root, "configs", "command.env"), []byte("TMPDIR='"+tmpDir+"'\nBINDIR='"+root+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origHTTP := taskHTTPDo
	origStart := startActionRun
	origInit := initctlRun
	defer func() {
		taskHTTPDo = origHTTP
		startActionRun = origStart
		initctlRun = origInit
	}()

	taskHTTPDo = func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == "https://example.com/gh/juewuy/ShellCrash@master/version" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("versionsh=1.0.0\n"))}, nil
		}
		t.Fatalf("unexpected URL requested: %s", req.URL.String())
		return nil, nil
	}
	startActionRun = func(crashDir, action string) error {
		t.Fatalf("start action should not be called: %s", action)
		return nil
	}
	initctlRun = func(opts initctl.Options) error {
		t.Fatal("initctl should not run when update is skipped")
		return nil
	}

	r := Runner{CrashDir: root}
	if err := r.Run([]string{"update_scripts"}); err != nil {
		t.Fatal(err)
	}
}

func makeTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func splitNonEmpty(s string) []string {
	out := []string{}
	curr := ""
	for _, r := range s {
		if r == '\n' {
			if curr != "" {
				out = append(out, curr)
			}
			curr = ""
			continue
		}
		curr += string(r)
	}
	if curr != "" {
		out = append(out, curr)
	}
	return out
}
