package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/testutil"
)

type fixture struct {
	in   Input
	vars map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{vars: map[string]string{}}
	dir := t.TempDir()
	f.in = Input{
		Keys:        keys.Env{GOOS: runtime.GOOS, Home: t.TempDir(), Getenv: func(k string) string { return f.vars[k] }},
		Dir:         dir,
		SecretsFile: filepath.Join(dir, "secrets.env"),
		LookPath:    func(string) (string, error) { return "", errors.New("not found") },
	}
	return f
}

// healthy sets up the test key, a matching .sops.yaml and the fixture file.
func (f *fixture) healthy(t *testing.T) {
	t.Helper()
	key := filepath.Join(t.TempDir(), "keys.txt")
	data, err := os.ReadFile(testutil.KeyFile())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, data, 0o600); err != nil {
		t.Fatal(err)
	}
	f.vars[keys.EnvKeyFile] = key
	if err := os.WriteFile(filepath.Join(f.in.Dir, ".sops.yaml"), []byte("creation_rules:\n  - age: "+testutil.PublicKey+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.CopyFixture(t, f.in.SecretsFile)
}

func byName(results []Result) map[string]Result {
	out := map[string]Result{}
	for _, r := range results {
		out[r.Name] = r
	}
	return out
}

func TestChecksRunInChainOrder(t *testing.T) {
	f := newFixture(t)
	f.healthy(t)
	var names []string
	for _, r := range Run(f.in) {
		names = append(names, r.Name)
	}
	want := "age key,key permissions,.sops.yaml,recipient,secrets file,decrypt,sops/age CLI"
	if got := strings.Join(names, ","); got != want {
		t.Fatalf("order %s, want %s", got, want)
	}
}

func TestHealthy(t *testing.T) {
	f := newFixture(t)
	f.healthy(t)
	results := Run(f.in)
	for _, r := range results {
		if r.Status != OK && r.Status != Info {
			t.Errorf("%s: %s %s", r.Name, r.Status, r.Detail)
		}
	}
	if Failed(results) {
		t.Fatal("Failed() on a healthy setup")
	}
	if got := byName(results)["decrypt"].Detail; got != "2 keys" {
		t.Errorf("decrypt detail %q, want only the key count", got)
	}
	if r := byName(results)["sops/age CLI"]; r.Status != Info || !strings.Contains(r.Detail, "not installed") {
		t.Errorf("tools: %+v", r)
	}
}

func TestNoKey(t *testing.T) {
	f := newFixture(t)
	r := byName(Run(f.in))
	if r["age key"].Status != Fail || r["age key"].Fix != "sopsy init" {
		t.Errorf("age key: %+v", r["age key"])
	}
	for _, name := range []string{"key permissions", "recipient", "decrypt"} {
		if r[name].Status != Skipped {
			t.Errorf("%s: %s, want skipped", name, r[name].Status)
		}
	}
}

func TestWorldReadableKeyWarns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes")
	}
	f := newFixture(t)
	f.healthy(t)
	key := f.vars[keys.EnvKeyFile]
	if err := os.Chmod(key, 0o644); err != nil {
		t.Fatal(err)
	}
	results := Run(f.in)
	r := byName(results)["key permissions"]
	if r.Status != Warn || r.Fix != "chmod 600 "+key {
		t.Fatalf("permissions: %+v", r)
	}
	if Failed(results) {
		t.Fatal("a warning made doctor fail")
	}
}

func TestNotARecipient(t *testing.T) {
	f := newFixture(t)
	f.healthy(t)
	other, err := keys.Generate(filepath.Join(t.TempDir(), "other.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f.vars[keys.EnvKeyFile] = other.Path
	results := Run(f.in)
	r := byName(results)
	if r["recipient"].Status != Fail || !strings.Contains(r["recipient"].Detail, other.Public) || !strings.Contains(r["recipient"].Fix, "sopsy pubkey") {
		t.Fatalf("recipient: %+v", r["recipient"])
	}
	if r["decrypt"].Status != Skipped {
		t.Errorf("decrypt: %s, want skipped", r["decrypt"].Status)
	}
	if !Failed(results) {
		t.Fatal("Failed() = false")
	}
}

func TestMissingConfigAndFile(t *testing.T) {
	f := newFixture(t)
	f.healthy(t)
	_ = os.Remove(filepath.Join(f.in.Dir, ".sops.yaml"))
	_ = os.Remove(f.in.SecretsFile)
	r := byName(Run(f.in))
	if r[".sops.yaml"].Status != Fail || r[".sops.yaml"].Fix != "sopsy init" {
		t.Errorf(".sops.yaml: %+v", r[".sops.yaml"])
	}
	if r["recipient"].Status != Skipped || r["decrypt"].Status != Skipped {
		t.Errorf("recipient %s, decrypt %s; want both skipped", r["recipient"].Status, r["decrypt"].Status)
	}
	if r["secrets file"].Status != Fail {
		t.Errorf("secrets file: %+v", r["secrets file"])
	}
}

func TestRecipientFromConfigWhenFileIsMissing(t *testing.T) {
	f := newFixture(t)
	f.healthy(t)
	_ = os.Remove(f.in.SecretsFile)
	r := byName(Run(f.in))["recipient"]
	if r.Status != OK || !strings.Contains(r.Detail, ".sops.yaml") {
		t.Fatalf("recipient: %+v", r)
	}
}

func TestTamperedFile(t *testing.T) {
	f := newFixture(t)
	f.healthy(t)
	data, err := os.ReadFile(f.in.SecretsFile)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, l := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(l, "DB_PASSWORD=") {
			kept = append(kept, l)
		}
	}
	if err := os.WriteFile(f.in.SecretsFile, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	r := byName(Run(f.in))["decrypt"]
	if r.Status != Fail || !strings.Contains(r.Detail, "modified outside SOPS") {
		t.Fatalf("decrypt: %+v", r)
	}
}

func TestToolsFound(t *testing.T) {
	f := newFixture(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f.in.LookPath = func(string) (string, error) { return self, nil }
	r := byName(Run(f.in))["sops/age CLI"]
	if r.Status != Info || !strings.Contains(r.Detail, self) {
		t.Fatalf("tools: %+v", r)
	}
}
