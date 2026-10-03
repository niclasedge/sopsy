package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/niclasedge/sopsy/internal/testutil"
)

func runWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	return w
}

func TestRunValueReachesChild(t *testing.T) {
	w := runWorld(t)
	r := w.runChild(append([]string{"run", "--"}, helperCmd("printenv", "API_TOKEN")...)...)
	r.wantCode(t, ExitOK)
	if r.stdout != testutil.FixtureValues["API_TOKEN"] {
		t.Fatal("child did not receive the decrypted value")
	}
	if r.stderr != "" {
		t.Fatalf("sopsy printed something itself: %q", r.stderr)
	}
}

func TestRunFileOverridesEnvironment(t *testing.T) {
	w := runWorld(t)
	w.vars["API_TOKEN"] = "old"
	r := w.runChild(append([]string{"run", "--"}, helperCmd("printenv", "API_TOKEN")...)...)
	r.wantCode(t, ExitOK)
	if r.stdout != testutil.FixtureValues["API_TOKEN"] {
		t.Fatal("the value from the file did not override the environment")
	}
}

func TestRunArgumentsVerbatim(t *testing.T) {
	w := runWorld(t)
	r := w.runChild(append([]string{"run", "--"}, helperCmd("args", "a b", "it's", "$HOME", "")...)...)
	r.wantCode(t, ExitOK)
	want := "\"a b\"\n\"it's\"\n\"$HOME\"\n\"\"\n"
	if r.stdout != want {
		t.Fatalf("child got\n%s\nwant\n%s", r.stdout, want)
	}
}

func TestRunWithoutDoubleDash(t *testing.T) {
	w := runWorld(t)
	r := w.runChild(append([]string{"run"}, helperCmd("exit", "0")...)...)
	r.wantCode(t, ExitOK)
}

func TestRunExitCodePassThrough(t *testing.T) {
	w := runWorld(t)
	r := w.runChild(append([]string{"run", "--"}, helperCmd("exit", "3")...)...)
	r.wantCode(t, 3)
}

func TestRunCommandNotFound(t *testing.T) {
	w := runWorld(t)
	r := w.runChild("run", "--", "sopsy-no-such-command")
	r.wantCode(t, 127)
	wantContains(t, r.stderr, "command not found")
}

func TestRunCommandNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows decides executability by extension")
	}
	w := runWorld(t)
	script := w.write("script.sh", "#!/bin/sh\nexit 0\n") // mode 0644: not executable
	r := w.runChild("run", "--", script)
	r.wantCode(t, 126)
}

func TestRunDecryptionFailsBeforeCommandLookup(t *testing.T) {
	w := runWorld(t)
	w.vars["SOPS_AGE_KEY_FILE"] = filepath.Join(w.home, "missing.txt")
	r := w.runChild("run", "--", "sopsy-no-such-command")
	r.wantCode(t, ExitFail)
}

func TestRunDoesNotStartChildWhenDecryptionFails(t *testing.T) {
	w := runWorld(t)
	w.write(DefaultFile, "API_TOKEN=ENC[broken]\nsops_version=3.13.3\n")
	marker := filepath.Join(w.home, "started")
	r := w.runChild(append([]string{"run", "--"}, helperCmd("marker", marker)...)...)
	r.wantCode(t, ExitFail)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("child was started")
	}
}

func TestRunMissingCommand(t *testing.T) {
	r := runWorld(t).run("run")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "missing command")
}
