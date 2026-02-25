package menuctl

import (
	"io"
	"strings"
	"testing"

	"shellcrash/internal/settingsctl"
)

func TestRunDispatchesSettingsAndStopFromMainMenu(t *testing.T) {
	origStart := menuRunStartAction
	origSettings := menuRunSettings
	defer func() {
		menuRunStartAction = origStart
		menuRunSettings = origSettings
	}()

	startCalls := []string{}
	menuRunStartAction = func(crashDir, action string, extraArgs ...string) error {
		startCalls = append(startCalls, action)
		return nil
	}

	settingsCalled := false
	menuRunSettings = func(opts settingsctl.Options, in io.Reader, out io.Writer) error {
		settingsCalled = true
		return nil
	}

	in := strings.NewReader("2\n3\n0\n")
	var out strings.Builder
	if err := Run(Options{CrashDir: "/tmp/sc", In: in, Out: &out, Err: &out}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !settingsCalled {
		t.Fatalf("expected settings menu to be called")
	}
	if len(startCalls) != 1 || startCalls[0] != "stop" {
		t.Fatalf("unexpected start action calls: %#v", startCalls)
	}
}

func TestRunPrintsInputErrorOnInvalidOption(t *testing.T) {
	in := strings.NewReader("x\n0\n")
	var out strings.Builder
	if err := Run(Options{CrashDir: "/tmp/sc", In: in, Out: &out, Err: &out}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(out.String(), "输入错误") {
		t.Fatalf("expected input error output, got=%q", out.String())
	}
}
