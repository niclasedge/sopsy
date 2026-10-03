package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
)

func TestNoArgumentsIsUsageError(t *testing.T) {
	r := newWorld(t).run()
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "Usage:")
}

func TestHelp(t *testing.T) {
	r := newWorld(t).run("help")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "sopsy init")
}

func TestUnknownCommand(t *testing.T) {
	r := newWorld(t).run("frobnicate")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, `unknown command "frobnicate"`, "sopsy help")
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	r := newWorld(t).run("init", "--nope")
	r.wantCode(t, ExitFail)
}

func TestNoKeyFoundExits125WithHint(t *testing.T) {
	var stderr bytes.Buffer
	err := &keys.NotFoundError{Searched: []string{"/a/keys.txt", "/b/keys.txt"}}
	code := report(Env{Stderr: &stderr}, err)
	if code != ExitFail {
		t.Fatalf("exit code %d, want %d", code, ExitFail)
	}
	wantContains(t, stderr.String(), "no age key found", "/a/keys.txt", "/b/keys.txt", "sopsy init")
}

func TestSecretsFileResolution(t *testing.T) {
	w := newWorld(t)
	env := w.env()
	if got, want := env.secretsFile(""), filepath.Join(w.dir, "secrets.env"); got != want {
		t.Errorf("default: %s, want %s", got, want)
	}
	w.vars[EnvFile] = "b.env"
	env = w.env()
	if got, want := env.secretsFile(""), filepath.Join(w.dir, "b.env"); got != want {
		t.Errorf("env: %s, want %s", got, want)
	}
	if got, want := env.secretsFile("a.env"), filepath.Join(w.dir, "a.env"); got != want {
		t.Errorf("flag over env: %s, want %s", got, want)
	}
}
