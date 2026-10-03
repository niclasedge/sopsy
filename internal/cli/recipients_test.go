package cli

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/store"
	"github.com/niclasedge/sopsy/internal/testutil"
)

func TestPubkey(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	r := w.run("pubkey")
	r.wantCode(t, ExitOK)
	if !regexp.MustCompile(`^age1[0-9a-z]+\n$`).MatchString(r.stdout) || r.stdout != testutil.PublicKey+"\n" {
		t.Fatalf("pubkey printed %q", r.stdout)
	}
}

func TestPubkeyWithoutKey(t *testing.T) {
	r := newWorld(t).run("pubkey")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "sopsy init")
}

func TestRecipientsListWithoutPrivateKey(t *testing.T) {
	w := newWorld(t)
	w.useFixture()
	r := w.run("recipients")
	r.wantCode(t, ExitOK)
	if r.stdout != testutil.PublicKey+"\n" {
		t.Fatalf("recipients printed %q", r.stdout)
	}
}

func TestRecipientsListMarksOwnKey(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	r := w.run("recipients")
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, testutil.PublicKey+"  (this machine)")
}

func secondKey(t *testing.T) *keys.Key {
	t.Helper()
	k, err := keys.Generate(filepath.Join(t.TempDir(), "second.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestRecipientsAddAndRemove(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	secrets := w.useFixture()
	second := secondKey(t)

	r := w.run("recipients", "add", second.Public)
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "added "+second.Public)
	wantContains(t, w.read(".sops.yaml"), second.Public)
	// The new machine can decrypt with sopsy and with the sops CLI.
	f, err := store.Load(secrets)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Decrypt(second.Identities, second.Public); err != nil {
		t.Fatalf("added key cannot decrypt: %v", err)
	}
	testutil.SopsDecrypt(t, secrets, second.Path)

	r = w.run("recipients", "add", second.Public)
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "already a recipient; nothing changed")

	r = w.run("recipients", "remove", second.Public)
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "removed "+second.Public, "WARNING: rotate every value", "  API_TOKEN\n", "  DB_PASSWORD\n")
	if strings.Contains(w.read(".sops.yaml"), second.Public) {
		t.Error(".sops.yaml still lists the removed key")
	}
	f, err = store.Load(secrets)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Decrypt(second.Identities, second.Public); err == nil {
		t.Fatal("removed key can still decrypt the new file")
	}
}

func TestRecipientsAddInvalidKeyChangesNothing(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	before, config := w.read(DefaultFile), w.read(".sops.yaml")
	r := w.run("recipients", "add", "age1notakey")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "not a valid age public key", "sopsy pubkey")
	if w.read(DefaultFile) != before || w.read(".sops.yaml") != config {
		t.Fatal("files changed")
	}
}

func TestRecipientsRemoveLastRefused(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	before, config := w.read(DefaultFile), w.read(".sops.yaml")
	r := w.run("recipients", "remove", testutil.PublicKey)
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, "last recipient", "nothing changed")
	if w.read(DefaultFile) != before || w.read(".sops.yaml") != config {
		t.Fatal("files changed")
	}
}

func TestRecipientsAddSyncsConfigOnly(t *testing.T) {
	// The file already lists the key (e.g. after `sops updatekeys`), only
	// .sops.yaml is behind; add brings it in line without touching the file.
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	other := secondKey(t)
	w.write(".sops.yaml", "creation_rules:\n  - age:\n      - "+other.Public+"\n")
	before := w.read(DefaultFile)
	r := w.run("recipients", "add", testutil.PublicKey)
	r.wantCode(t, ExitOK)
	if w.read(DefaultFile) != before {
		t.Error("secrets file changed although it already listed the key")
	}
	wantContains(t, w.read(".sops.yaml"), testutil.PublicKey, other.Public)
}

func TestRecipientsRemoveUnknown(t *testing.T) {
	w := newWorld(t)
	w.useTestKey()
	w.useFixture()
	r := w.run("recipients", "remove", secondKey(t).Public)
	r.wantCode(t, ExitOK)
	wantContains(t, r.stdout, "not a recipient; nothing changed")
}

func TestRecipientsUnknownSubcommand(t *testing.T) {
	w := newWorld(t)
	w.useFixture()
	r := w.run("recipients", "frob")
	r.wantCode(t, ExitFail)
	wantContains(t, r.stderr, `unknown recipients subcommand "frob"`)
}
