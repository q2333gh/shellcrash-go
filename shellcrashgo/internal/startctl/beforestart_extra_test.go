package startctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckNetworkReachability(t *testing.T) {
	oldHas := beforeStartHasCommand
	oldPing := beforeStartPing
	oldSleep := beforeStartSleep
	defer func() {
		beforeStartHasCommand = oldHas
		beforeStartPing = oldPing
		beforeStartSleep = oldSleep
	}()

	beforeStartHasCommand = func(name string) bool { return name == "ping" }
	beforeStartSleep = func(_ time.Duration) {}
	callTargets := make([]string, 0, 4)
	beforeStartPing = func(host string) bool {
		callTargets = append(callTargets, host)
		return host == "dns.alidns.com"
	}

	ctl := Controller{
		Cfg: Config{
			TmpDir:       t.TempDir(),
			NetworkCheck: "ON",
		},
	}
	if err := ctl.checkNetworkReachability(); err != nil {
		t.Fatal(err)
	}
	if len(callTargets) != 3 {
		t.Fatalf("expected 3 ping attempts before success, got %d", len(callTargets))
	}
}

func TestCheckNetworkReachabilitySkippedWhenDisabled(t *testing.T) {
	oldHas := beforeStartHasCommand
	oldPing := beforeStartPing
	defer func() {
		beforeStartHasCommand = oldHas
		beforeStartPing = oldPing
	}()

	beforeStartHasCommand = func(string) bool { return true }
	called := false
	beforeStartPing = func(string) bool {
		called = true
		return false
	}

	ctl := Controller{
		Cfg: Config{
			TmpDir:       t.TempDir(),
			NetworkCheck: "OFF",
		},
	}
	if err := ctl.checkNetworkReachability(); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatalf("ping should not be called when network_check=OFF")
	}
}

func TestRewriteUserFilesForShellCrash(t *testing.T) {
	td := t.TempDir()
	passwdPath := filepath.Join(td, "passwd")
	groupPath := filepath.Join(td, "group")
	passwdSrc := "root:x:0:0:root:/root:/bin/sh\nold:x:1000:1000:::\nshellcrash:x:1001:7890:::\n"
	groupSrc := "root:x:0:\nshellcrash:x:7890:\nusers:x:100:\n"
	if err := os.WriteFile(passwdPath, []byte(passwdSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(groupPath, []byte(groupSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteUserFilesForShellCrash(passwdPath, groupPath); err != nil {
		t.Fatal(err)
	}

	passwdOut, err := os.ReadFile(passwdPath)
	if err != nil {
		t.Fatal(err)
	}
	groupOut, err := os.ReadFile(groupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(passwdOut), "shellcrash:x:0:7890:::") {
		t.Fatalf("shellcrash passwd entry missing: %s", string(passwdOut))
	}
	if strings.Contains(string(passwdOut), "shellcrash:x:1001:7890:::") {
		t.Fatalf("stale shellcrash passwd entry should be removed: %s", string(passwdOut))
	}
	if !strings.Contains(string(groupOut), "shellcrash:x:7890:") {
		t.Fatalf("shellcrash group entry missing: %s", string(groupOut))
	}
}

func TestEnsureTunModule(t *testing.T) {
	oldHas := beforeStartHasCommand
	oldModprobe := beforeStartModprobeTun
	defer func() {
		beforeStartHasCommand = oldHas
		beforeStartModprobeTun = oldModprobe
	}()

	beforeStartHasCommand = func(name string) bool { return name == "modprobe" }
	called := false
	beforeStartModprobeTun = func() {
		called = true
	}

	ctl := Controller{Cfg: Config{RedirMod: "Mix"}}
	ctl.ensureTunModule()
	if !called {
		t.Fatalf("expected modprobe tun call")
	}
}
