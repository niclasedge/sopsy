package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/testutil"
)

func TestValidateRecipient(t *testing.T) {
	if err := ValidateRecipient(testutil.PublicKey); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "age1nope", "AGE-SECRET-KEY-1ABC", "hello"} {
		if err := ValidateRecipient(bad); err == nil {
			t.Errorf("ValidateRecipient(%q) accepted", bad)
		}
	}
}

func TestAddAndRemoveRecipient(t *testing.T) {
	p := fixture(t)
	own := testKey(t)
	second, err := keys.Generate(filepath.Join(t.TempDir(), "second.txt"))
	if err != nil {
		t.Fatal(err)
	}
	valueLines := func() []string {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, l := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(l, "sops_") {
				out = append(out, l)
			}
		}
		return out
	}
	before := strings.Join(valueLines(), "\n")

	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.ChangeRecipients(own.Identities, own.Public, []string{second.Public}, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Join(valueLines(), "\n") != before {
		t.Error("adding a recipient changed value ciphertexts")
	}
	f, err = Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Decrypt(second.Identities, second.Public); err != nil {
		t.Fatalf("second key cannot decrypt after add: %v", err)
	}
	testutil.SopsDecrypt(t, p, second.Path)

	f, err = Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.ChangeRecipients(own.Identities, own.Public, nil, []string{second.Public}); err != nil {
		t.Fatal(err)
	}
	f, err = Load(p)
	if err != nil {
		t.Fatal(err)
	}
	var nr *NotRecipientError
	if _, err := f.Decrypt(second.Identities, second.Public); !errors.As(err, &nr) {
		t.Fatalf("second key still decrypts after remove: %v", err)
	}
	if got := f.Recipients(); len(got) != 1 || got[0] != own.Public {
		t.Fatalf("recipients after remove: %v", got)
	}
}

func TestRemoveLastRecipientRefused(t *testing.T) {
	p := fixture(t)
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	own := testKey(t)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.ChangeRecipients(own.Identities, own.Public, nil, []string{own.Public}); !errors.Is(err, ErrLastRecipient) {
		t.Fatalf("want ErrLastRecipient, got %v", err)
	}
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("file changed")
	}
}

func TestChangeRecipientsRequiresDecryption(t *testing.T) {
	p := fixture(t)
	other, err := keys.Generate(filepath.Join(t.TempDir(), "other.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	err = f.ChangeRecipients(other.Identities, other.Public, []string{other.Public}, nil)
	var nr *NotRecipientError
	if !errors.As(err, &nr) {
		t.Fatalf("a non-recipient added itself: %v", err)
	}
}
