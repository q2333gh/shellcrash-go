package taskctl

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEnsureUserTaskFileCreatesHeader(t *testing.T) {
	root := t.TempDir()
	if err := ensureUserTaskFile(root); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "task", "task.user"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 || b[0] != '#' {
		t.Fatalf("unexpected task.user content: %q", string(b))
	}
}

func TestSetCronTaskPersistsAndCronsets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}
	line := "0 3 * * * " + filepath.Join(root, "task", "task.sh") + " 103 自动更新订阅"

	orig := startActionRunWithArgs
	defer func() { startActionRunWithArgs = orig }()

	called := false
	startActionRunWithArgs = func(crashDir, action string, extraArgs []string) error {
		called = true
		if crashDir != root || action != "cronset" {
			t.Fatalf("unexpected call: %s %s", crashDir, action)
		}
		if len(extraArgs) != 2 || extraArgs[0] != "自动更新订阅" || extraArgs[1] != line {
			t.Fatalf("unexpected cron args: %v", extraArgs)
		}
		return nil
	}

	if err := setCronTask(root, "自动更新订阅", line); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected cronset call")
	}
	b, err := os.ReadFile(filepath.Join(root, "task", "cron"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != line+"\n" {
		t.Fatalf("unexpected cron persistence: %q", got)
	}
}

func TestRemoveTaskByKeywordPrunesTaskFiles(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cron", "bfstart", "afstart", "running", "affirewall"} {
		content := "keep line\nremove-me line\n"
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	orig := startActionRunWithArgs
	defer func() { startActionRunWithArgs = orig }()
	startActionRunWithArgs = func(crashDir, action string, extraArgs []string) error {
		if crashDir != root || action != "cronset" {
			t.Fatalf("unexpected call: %s %s", crashDir, action)
		}
		if !reflect.DeepEqual(extraArgs, []string{"remove-me"}) {
			t.Fatalf("unexpected cronset args: %v", extraArgs)
		}
		return nil
	}

	if err := removeTaskByKeyword(root, "remove-me"); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"cron", "bfstart", "afstart", "running", "affirewall"} {
		b, err := os.ReadFile(filepath.Join(taskDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if got := string(b); got != "keep line\n" {
			t.Fatalf("unexpected %s content: %q", name, got)
		}
	}
}

func TestCollectManagedTasksParsesEntries(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "cron"), []byte("0 3 * * * /x/task/task.sh 103 自动更新订阅\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "afstart"), []byte("/x/task/task.sh 107 启动后任务\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := collectManagedTasks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected managed task count: %d", len(items))
	}
	if items[0].ID != "103" && items[1].ID != "103" {
		t.Fatalf("missing cron task parse: %+v", items)
	}
	if items[0].ID != "107" && items[1].ID != "107" {
		t.Fatalf("missing afstart task parse: %+v", items)
	}
}

func TestApplyRecommendedTasksWritesDefaultEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "task"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := startActionRunWithArgs
	defer func() { startActionRunWithArgs = orig }()
	startActionRunWithArgs = func(crashDir, action string, extraArgs []string) error {
		if crashDir != root || action != "cronset" {
			t.Fatalf("unexpected call: %s %s", crashDir, action)
		}
		if len(extraArgs) != 2 || extraArgs[0] != "自动更新订阅" {
			t.Fatalf("unexpected cron args: %v", extraArgs)
		}
		return nil
	}

	if err := ApplyRecommendedTasks(root); err != nil {
		t.Fatal(err)
	}

	running, err := os.ReadFile(filepath.Join(root, "task", "running"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(running), " 106 订阅保活") {
		t.Fatalf("expected running recommended task, got: %s", string(running))
	}

	afstart, err := os.ReadFile(filepath.Join(root, "task", "afstart"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(afstart), " 107 重启后保存面板") {
		t.Fatalf("expected afstart recommended task, got: %s", string(afstart))
	}

	cron, err := os.ReadFile(filepath.Join(root, "task", "cron"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cron), " 103 自动更新订阅") {
		t.Fatalf("expected cron recommended task, got: %s", string(cron))
	}
}
