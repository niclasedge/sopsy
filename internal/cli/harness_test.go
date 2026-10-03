package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/testutil"
)

// world is an isolated machine for one test: its own home directory, project
// directory and environment variables. Nothing of the developer's real setup
// leaks in.
type world struct {
	t     *testing.T
	home  string
	dir   string
	vars  map[string]string
	stdin string
}

type result struct {
	stdout, stderr string
	code           int
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{t: t, home: t.TempDir(), dir: t.TempDir(), vars: map[string]string{}}
	if runtime.GOOS == "windows" {
		w.vars["AppData"] = filepath.Join(w.home, "AppData", "Roaming")
	}
	return w
}

func (w *world) env() Env {
	getenv := func(k string) string { return w.vars[k] }
	return Env{
		Stdin:  strings.NewReader(w.stdin),
		Dir:    w.dir,
		Getenv: getenv,
		Keys:   keys.Env{GOOS: runtime.GOOS, Home: w.home, Getenv: getenv},
	}
}

// run executes sopsy in the world and fails the test if any known secret
// value appears in its output.
func (w *world) run(args ...string) result {
	w.t.Helper()
	var stdout, stderr bytes.Buffer
	env := w.env()
	env.Stdout, env.Stderr = &stdout, &stderr
	code := Run(env, args)
	r := result{stdout: stdout.String(), stderr: stderr.String(), code: code}
	testutil.AssertNoSecret(w.t, r.stdout, r.stderr)
	return r
}

// useTestKey points the world at the committed test-only key.
func (w *world) useTestKey() {
	w.vars[keys.EnvKeyFile] = testutil.KeyFile()
}

func (w *world) write(name, content string) string {
	w.t.Helper()
	p := filepath.Join(w.dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		w.t.Fatal(err)
	}
	return p
}

func (w *world) read(name string) string {
	w.t.Helper()
	data, err := os.ReadFile(filepath.Join(w.dir, name))
	if err != nil {
		w.t.Fatal(err)
	}
	return string(data)
}

func (r result) wantCode(t *testing.T, code int) {
	t.Helper()
	if r.code != code {
		t.Fatalf("exit code %d, want %d\nstdout:\n%s\nstderr:\n%s", r.code, code, r.stdout, r.stderr)
	}
}

func wantContains(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(out, p) {
			t.Errorf("output does not contain %q:\n%s", p, out)
		}
	}
}
