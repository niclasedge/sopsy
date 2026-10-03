package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsops/sops/v3/config"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/testutil"
)

func testKey(t *testing.T) *keys.Key {
	t.Helper()
	k, err := keys.LoadFile(testutil.KeyFile())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func fixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "secrets.env")
	testutil.CopyFixture(t, p)
	return p
}

func TestDecryptSopsCLIFixture(t *testing.T) {
	k := testKey(t)
	f, err := Load(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Recipients(); len(got) != 1 || got[0] != testutil.PublicKey {
		t.Fatalf("Recipients() = %v", got)
	}
	plain, err := f.Decrypt(k.Identities, k.Public)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range plain.Entries() {
		got[e.Name] = e.Value
	}
	for name, want := range testutil.FixtureValues {
		if got[name] != want {
			t.Errorf("%s: wrong value", name)
		}
	}
}

func TestCRLFFileDecrypts(t *testing.T) {
	p := fixture(t)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	crlf := strings.ReplaceAll(string(data), "\n", "\r\n")
	if err := os.WriteFile(p, []byte(crlf), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	k := testKey(t)
	plain, err := f.Decrypt(k.Identities, k.Public)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range plain.Entries() {
		if e.Value != testutil.FixtureValues[e.Name] {
			t.Errorf("%s: wrong value", e.Name)
		}
	}
}

func TestNamesWithoutDecrypting(t *testing.T) {
	f, err := Load(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.Names(), ","); got != "API_TOKEN,DB_PASSWORD" {
		t.Fatalf("Names() = %s", got)
	}
}

func TestNotARecipient(t *testing.T) {
	other, err := keys.Generate(filepath.Join(t.TempDir(), "keys.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Load(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Decrypt(other.Identities, other.Public)
	var nr *NotRecipientError
	if !errors.As(err, &nr) || nr.Public != other.Public {
		t.Fatalf("want NotRecipientError naming %s, got %v", other.Public, err)
	}
}

func TestTamperedFile(t *testing.T) {
	p := fixture(t)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	// Drop one entry: every remaining ciphertext is valid, only the MAC breaks.
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "DB_PASSWORD=") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(p, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	k := testKey(t)
	_, err = f.Decrypt(k.Identities, k.Public)
	var te *TamperedError
	if !errors.As(err, &te) {
		t.Fatalf("want TamperedError, got %v", err)
	}
}

func TestMalformedLineHidesContent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secrets.env")
	content := "GOOD=ENC[x]\ns3cr3t-test-value\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Line != 2 {
		t.Fatalf("want ParseError on line 2, got %v", err)
	}
	testutil.AssertNoSecret(t, err.Error())
}

func TestPlainFileIsRejected(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(p, []byte("A=b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("want ErrNotEncrypted, got %v", err)
	}
}

func createEmpty(t *testing.T, recipient string) string {
	t.Helper()
	dir := t.TempDir()
	secrets := filepath.Join(dir, "secrets.env")
	conf := filepath.Join(dir, sopsconfig.FileName)
	if err := sopsconfig.Create(conf, secrets, recipient); err != nil {
		t.Fatal(err)
	}
	rule, err := config.LoadCreationRuleForFile(conf, secrets, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := Create(secrets, rule); err != nil {
		t.Fatal(err)
	}
	return secrets
}

func TestCreateEmptyFile(t *testing.T) {
	k := testKey(t)
	p := createEmpty(t, k.Public)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := f.Decrypt(k.Identities, k.Public)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(plain.Entries()); n != 0 {
		t.Fatalf("new file has %d entries", n)
	}
	rule, _ := config.LoadCreationRuleForFile(filepath.Join(filepath.Dir(p), sopsconfig.FileName), p, nil)
	if err := Create(p, rule); err == nil {
		t.Fatal("Create overwrote an existing file")
	}
}

func TestCreatedFileDecryptsWithSopsCLI(t *testing.T) {
	p := createEmpty(t, testutil.PublicKey)
	if out := testutil.SopsDecrypt(t, p, testutil.KeyFile()); strings.TrimSpace(out) != "" {
		t.Fatalf("sops decrypt of empty file returned %q", out)
	}
}

func TestCreateLeavesNoTempFiles(t *testing.T) {
	p := createEmpty(t, testutil.PublicKey)
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}
