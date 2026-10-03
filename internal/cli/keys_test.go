package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/store"
	"github.com/niclasedge/sopsy/internal/testutil"
)

const testValue = "s3cr3t-test-value"

// values decrypts the world's secrets file with the test key.
func (w *world) values() map[string]string {
	w.t.Helper()
	f, err := store.Load(filepath.Join(w.dir, DefaultFile))
	if err != nil {
		w.t.Fatal(err)
	}
	k, err := keys.LoadFile(testutil.KeyFile())
	if err != nil {
		w.t.Fatal(err)
	}
	plain, err := f.Decrypt(k.Identities, k.Public)
	if err != nil {
		w.t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range plain.Entries() {
		out[e.Name] = e.Value
	}
	return out
}

func TestSetPiped(t *testing.T) {
	tests := []struct{ name, stdin, want string }{
		{"no newline", testValue, testValue},
		{"one newline stripped", testValue + "\n", testValue},
		{"crlf stripped", testValue + "\r\n", testValue},
		{"only one newline stripped", testValue + "\n\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t)
			w.useTestKey()
			w.useFixture()
			w.stdin = tt.stdin
			r := w.run("set", "NEW_KEY")
			if tt.want == "" {
				// "value\n\n" keeps one newline, which is rejected.
				r.wantCode(t, ExitFail)
				wantContains(t, r.stderr, "line break", "nothing was written")
				return
			}
			r.wantCode(t, ExitOK)
			wantContains(t, r.stdout, "added NEW_KEY")
			if got := w.values()["NEW_KEY"]; got != tt.want {
				t.Fatal("stored value differs from the piped value")
			}
		})
	}
}

func TestSetReplacesAndKeepsOtherKeys(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	w.stdin = testValue
	w.run("set", "API_TOKEN").wantCode(t, ExitOK)
	got := w.values()
	if got["API_TOKEN"] != testValue || got["DB_PASSWORD"] != testutil.FixtureValues["DB_PASSWORD"] {
		t.Fatal("set changed the wrong keys")
	}
	out := testutil.SopsDecrypt(t, filepath.Join(w.dir, DefaultFile), testutil.KeyFile())
	if !strings.Contains(out, "API_TOKEN="+testValue+"\n") || !strings.Contains(out, "DB_PASSWORD=") {
		t.Fatal("sops CLI does not see the new value")
	}
}

func TestSetInteractive(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	var prompts []string
	w.prompt = func(p string) (string, error) {
		prompts = append(prompts, p)
		return testValue, nil
	}
	w.run("set", "NEW_KEY").wantCode(t, ExitOK)
	if len(prompts) != 2 {
		t.Fatalf("want a prompt and a confirmation, got %q", prompts)
	}
	if w.values()["NEW_KEY"] != testValue {
		t.Fatal("value not stored")
	}
}

func TestSetInteractiveMismatch(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	before := w.read(filepath.Base(w.useFixture()))
	entries := []string{testValue, "second-s3cr3t-test-value"}
	w.prompt = func(string) (string, error) {
		v := entries[0]
		entries = entries[1:]
		return v, nil
	}
	r := w.run("set", "NEW_KEY")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "do not match")
	if w.read(DefaultFile) != before {
		t.Fatal("file changed")
	}
}

func TestSetRefusesValueArgument(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	before := w.read(filepath.Base(w.useFixture()))
	r := w.run("set", "API_TOKEN", testValue)
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "never from arguments", "stdin", "prompt")
	if w.read(DefaultFile) != before {
		t.Fatal("file changed")
	}
}

func TestSetRejectsEmptyValueAndBadName(t *testing.T) {
	for _, args := range [][]string{{"set", "EMPTY"}, {"set", "1BAD-NAME"}, {"set", "sops_mac"}} {
		w := newWorld(t)
		w.useTestKey()
		before := w.read(filepath.Base(w.useFixture()))
		w.stdin = "\n"
		w.run(args...).wantCode(t, ExitFail)
		if w.read(DefaultFile) != before {
			t.Fatalf("%v changed the file", args)
		}
	}
}

func TestSetDetectsConcurrentChange(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	p := w.useFixture()
	other := "changed by another process\n"
	// The prompt runs after the file was read and decrypted; another
	// process writes the file while the user types.
	w.prompt = func(string) (string, error) {
		if err := os.WriteFile(p, []byte(other), 0o644); err != nil {
			t.Fatal(err)
		}
		return testValue, nil
	}
	r := w.run("set", "NEW_KEY")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "changed on disk", "nothing was written", "again")
	if w.read(DefaultFile) != other {
		t.Fatal("sopsy overwrote the concurrent change")
	}
}

func TestUnset(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	r := w.run("unset", "API_TOKEN")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "removed API_TOKEN")
	out := testutil.SopsDecrypt(t, filepath.Join(w.dir, DefaultFile), testutil.KeyFile())
	if strings.Contains(out, "API_TOKEN") || !strings.Contains(out, "DB_PASSWORD=") {
		t.Fatal("unset removed the wrong keys")
	}

	r = w.run("unset", "API_TOKEN")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "nothing removed")
}

func TestKeysWithoutPrivateKey(t *testing.T) {
	w := newWorld(t)
	w.useFixture()
	w.write(DefaultFile, strings.Replace(w.read(DefaultFile), "API_TOKEN=", "ZED=", 1))
	r := w.run("keys")
	r.wantCode(t, ExitOK)
	if r.stdout != "DB_PASSWORD\nZED\n" {
		t.Fatalf("keys printed %q, want sorted names", r.stdout)
	}
}

func TestKeysMissingFile(t *testing.T) {
	w := newWorld(t)
	r := w.run("keys")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "does not exist", "sopsy init")
}

func TestSetNotARecipient(t *testing.T) {
	w := newWorld(t)
	w.useFixture()
	w.run("init").wantCode(t, ExitOK) // creates a fresh key that is not a recipient
	w.stdin = testValue
	r := w.run("set", "NEW_KEY")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "is not a recipient", "sopsy pubkey")
}
