package installpathctl

import (
	"os"
	"strings"
	"testing"
)

func TestRunSelectDefaultEtc(t *testing.T) {
	in := strings.NewReader("1\n1\n")
	res, err := RunSelect(Options{In: in, Out: os.Stdout})
	if err != nil {
		t.Fatalf("run select: %v", err)
	}
	if res.Dir != "/etc" {
		t.Fatalf("unexpected dir: %q", res.Dir)
	}
	if res.CrashDir != "/etc/ShellCrash" {
		t.Fatalf("unexpected crashdir: %q", res.CrashDir)
	}
}

func TestRunCustomRejectsTmpPath(t *testing.T) {
	custom := "/var"
	if st, err := os.Stat(custom); err != nil || !st.IsDir() {
		t.Fatalf("expected %s directory available", custom)
	}
	in := strings.NewReader("/tmp/demo\n" + custom + "\n")
	res, err := RunCustom(Options{In: in, Out: os.Stdout})
	if err != nil {
		t.Fatalf("run custom: %v", err)
	}
	if res.Dir != custom {
		t.Fatalf("unexpected custom dir: %q", res.Dir)
	}
}
