package startctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenusTGBotScriptDispatchesToGoBotBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "bot_tg.sh")
	runTGBotWrapperDispatchTest(t, script)
}

func TestToolsTGBotScriptDispatchesToGoBotBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "tools", "tg_bot.sh")
	runTGBotWrapperDispatchTest(t, script)
}

func TestMenusTGBotBindScriptGetChatIDDispatchesToGoBotBinary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "bot_tg_bind.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatalf("mkdir crashdir: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	outChat := filepath.Join(td, "chat.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeBot := filepath.Join(fakeBin, "shellcrash-tgbot")
	body := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"$SC_MARKER\"\n" +
		"printf '12345678'\n"
	if err := os.WriteFile(fakeBot, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake bot binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; public_key=test-key; url_tg='https://example.test/getUpdates'; get_chatid; printf '%s' \"$chat_ID\" > \"$SC_CHAT\"")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"SC_CHAT="+outChat,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run bot_tg_bind wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	want := "--crashdir " + crashDir + " bind-chatid --url https://example.test/getUpdates --key test-key"
	if got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
	chatBytes, err := os.ReadFile(outChat)
	if err != nil {
		t.Fatalf("read chat output: %v", err)
	}
	if strings.TrimSpace(string(chatBytes)) != "12345678" {
		t.Fatalf("unexpected chat id value: %q", strings.TrimSpace(string(chatBytes)))
	}
}

func TestMenusTGBotServiceScriptDispatchesToGoStartctl(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	script := filepath.Join(repoRoot, "scripts", "menus", "bot_tg_service.sh")

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatalf("mkdir crashdir: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeStartctl := filepath.Join(fakeBin, "shellcrash-startctl")
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeStartctl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake startctl binary: %v", err)
	}

	cmd := exec.Command("sh", "-c", ". \"$SC_SCRIPT\"; bot_tg_start; bot_tg_stop; bot_tg_cron")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_SCRIPT="+script,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run bot_tg_service wrapper: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(gotBytes)), "\n")
	want := []string{"bot_tg_start", "bot_tg_stop", "bot_tg_cron"}
	if len(got) != len(want) {
		t.Fatalf("unexpected call count: got=%d want=%d values=%q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected call at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func runTGBotWrapperDispatchTest(t *testing.T, scriptPath string) {
	t.Helper()

	td := t.TempDir()
	crashDir := filepath.Join(td, "ShellCrash")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatalf("mkdir crashdir: %v", err)
	}

	marker := filepath.Join(td, "marker.txt")
	fakeBin := filepath.Join(td, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakeBot := filepath.Join(fakeBin, "shellcrash-tgbot")
	body := "#!/bin/sh\nprintf '%s' \"$*\" > \"$SC_MARKER\"\n"
	if err := os.WriteFile(fakeBot, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake bot binary: %v", err)
	}

	cmd := exec.Command("sh", scriptPath, "foo", "bar")
	cmd.Env = append(os.Environ(),
		"CRASHDIR="+crashDir,
		"SC_MARKER="+marker,
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run wrapper direct-exec: %v, out=%s", err, string(out))
	}

	gotBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	want := "--crashdir " + crashDir + " foo bar"
	if got != want {
		t.Fatalf("unexpected dispatch args: got=%q want=%q", got, want)
	}
}
