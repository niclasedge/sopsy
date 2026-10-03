package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/testutil"
)

func TestInitFreshMachine(t *testing.T) {
	w := newWorld(t)
	r := w.run("init")
	r.wantCode(t, ExitOK)

	lines := strings.Split(strings.TrimSpace(r.stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("want one line per step, got:\n%s", r.stdout)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "created") {
			t.Errorf("want created, got %q", l)
		}
	}
	pub := regexp.MustCompile(`age1[0-9a-z]+`).FindString(lines[0])
	if pub == "" {
		t.Fatalf("public key not printed: %q", lines[0])
	}

	keyPath, err := w.env().Keys.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key not created at the SOPS default location: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode %o, want 600", info.Mode().Perm())
	}
	wantContains(t, w.read(".sops.yaml"), pub)
	if out := testutil.SopsDecrypt(t, filepath.Join(w.dir, "secrets.env"), keyPath); strings.TrimSpace(out) != "" {
		t.Errorf("new secrets file is not empty: %q", out)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	w := newWorld(t)
	w.run("init").wantCode(t, ExitOK)
	keyPath, _ := w.env().Keys.DefaultPath()
	files := []string{keyPath, filepath.Join(w.dir, ".sops.yaml"), filepath.Join(w.dir, "secrets.env")}
	before := mtimes(t, files)
	time.Sleep(20 * time.Millisecond)

	r := w.run("init")
	r.wantCode(t, ExitOK)
	lines := strings.Split(strings.TrimSpace(r.stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("want three lines, got:\n%s", r.stdout)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "present") {
			t.Errorf("want present, got %q", l)
		}
	}
	after := mtimes(t, files)
	for i, f := range files {
		if !before[i].Equal(after[i]) {
			t.Errorf("%s was modified by the second run", f)
		}
	}
}

func mtimes(t *testing.T, files []string) []time.Time {
	t.Helper()
	var out []time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, info.ModTime())
	}
	return out
}

func TestInitKeepsExistingKeyAndConfig(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	// The config lives in a parent directory and does not list the own key.
	other, err := keys.Generate(filepath.Join(t.TempDir(), "other.txt"))
	if err != nil {
		t.Fatal(err)
	}
	config := "# team config\ncreation_rules:\n  - age: " + other.Public + "\n"
	w.write(".sops.yaml", config)
	sub := filepath.Join(w.dir, "app")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	w.dir = sub

	r := w.run("init")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "present  age key", testutil.KeyFile(), "present  .sops.yaml", "own key is NOT a recipient", "created  secrets file")
	if got := w.read("../.sops.yaml"); got != config {
		t.Errorf(".sops.yaml was modified:\n%s", got)
	}
}

func TestInitReportsOwnKeyAsRecipient(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.write(".sops.yaml", "creation_rules:\n  - path_regex: secrets\\.env$\n    age: "+testutil.PublicKey+"\n")
	r := w.run("init")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "own key is a recipient")
}

func TestInitWithExplicitMissingKeyFails(t *testing.T) {
	w := newWorld(t)
	missing := filepath.Join(w.home, "nope.txt")
	w.vars[keys.EnvKeyFile] = missing
	r := w.run("init")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, missing, "unset "+keys.EnvKeyFile)
	if _, err := os.Stat(filepath.Join(w.dir, ".sops.yaml")); err == nil {
		t.Error("init continued after the key step failed")
	}
}

func TestInitCustomFile(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.run("init", "--file", "prod.env").wantCode(t, ExitOK)
	if _, err := os.Stat(filepath.Join(w.dir, "prod.env")); err != nil {
		t.Fatal(err)
	}
	wantContains(t, w.read(".sops.yaml"), `prod\.env`)
}

func TestInitNoMatchingRule(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.write(".sops.yaml", "creation_rules:\n  - path_regex: other\\.env$\n    age: "+testutil.PublicKey+"\n")
	r := w.run("init")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "no creation rule")
}
