package cli

import (
	"bytes"
	"errors"
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
	// prompt, when set, makes stdin a terminal: each hidden-prompt read
	// calls it with the prompt text.
	prompt func(string) (string, error)
	// browser receives the URL `sopsy ui` opens; tests never start a real
	// browser.
	browser func(string) error
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
		Stdin:      strings.NewReader(w.stdin),
		Dir:        w.dir,
		Getenv:     getenv,
		Environ:    w.environ,
		Keys:       keys.Env{GOOS: runtime.GOOS, Home: w.home, Getenv: getenv},
		ReadSecret: w.prompt,
		OpenBrowser: func(url string) error {
			if w.browser == nil {
				return errors.New("no browser in tests")
			}
			return w.browser(url)
		},
	}
}

// environ is the real environment without anything SOPS- or sopsy-related,
// plus the world's variables.
func (w *world) environ() []string {
	var out []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "SOPS") {
			out = append(out, kv)
		}
	}
	for k, v := range w.vars {
		out = append(out, k+"="+v)
	}
	return out
}

// run executes sopsy in the world and fails the test if any known secret
// value appears in its output.
func (w *world) run(args ...string) result {
	w.t.Helper()
	r := w.exec(args)
	testutil.AssertNoSecret(w.t, r.stdout, r.stderr)
	return r
}

// runChild executes a sopsy command that starts a child. The child's stdout
// may contain secrets on purpose; sopsy's own stderr must not.
func (w *world) runChild(args ...string) result {
	w.t.Helper()
	r := w.exec(args)
	testutil.AssertNoSecret(w.t, r.stderr)
	return r
}

func (w *world) exec(args []string) result {
	var stdout, stderr bytes.Buffer
	env := w.env()
	env.Stdout, env.Stderr = &stdout, &stderr
	code := Run(env, args)
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// useTestKey points the world at the committed test-only key.
func (w *world) useTestKey() {
	w.vars[keys.EnvKeyFile] = testutil.KeyFile()
}

// useFixture copies the sops-CLI-encrypted fixture to the default secrets
// file and writes a matching .sops.yaml.
func (w *world) useFixture() string {
	w.t.Helper()
	p := filepath.Join(w.dir, DefaultFile)
	testutil.CopyFixture(w.t, p)
	w.write(".sops.yaml", "creation_rules:\n  - age: "+testutil.PublicKey+"\n")
	return p
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
