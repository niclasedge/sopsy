package store

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/niclasedge/sopsy/internal/testutil"
)

func TestValidateName(t *testing.T) {
	tests := map[string]bool{
		"API_TOKEN": true,
		"_x":        true,
		"a1":        true,
		"1BAD":      false,
		"BAD-NAME":  false,
		"":          false,
		"A B":       false,
		"sops_mac":  false,
		"SOPS_OK":   true,
		"ÄPFEL":     false,
	}
	for name, ok := range tests {
		if err := ValidateName(name); (err == nil) != ok {
			t.Errorf("ValidateName(%q) = %v, want ok=%v", name, err, ok)
		}
	}
}

func TestValidateValue(t *testing.T) {
	tests := map[string]bool{
		"plain":            true,
		"with spaces = ok": true,
		"":                 false,
		"two\nlines":       false,
		"carriage\rreturn": false,
	}
	for value, ok := range tests {
		if err := ValidateValue(value); (err == nil) != ok {
			t.Errorf("ValidateValue(%q) = %v, want ok=%v", value, err, ok)
		}
	}
}

func decryptFixture(t *testing.T) (string, *Plain) {
	t.Helper()
	p := fixture(t)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	k := testKey(t)
	plain, err := f.Decrypt(k.Identities, k.Public)
	if err != nil {
		t.Fatal(err)
	}
	return p, plain
}

func TestSetUnsetSaveKeepsSopsCompatibility(t *testing.T) {
	p, plain := decryptFixture(t)
	before, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if existed, err := plain.Set("NEW_KEY", "s3cr3t-test-value"); err != nil || existed {
		t.Fatalf("Set new = %v, %v", existed, err)
	}
	if existed, err := plain.Set("API_TOKEN", "second-s3cr3t-test-value"); err != nil || !existed {
		t.Fatalf("Set existing = %v, %v", existed, err)
	}
	if !plain.Unset("DB_PASSWORD") || plain.Unset("MISSING") {
		t.Fatal("Unset reported the wrong result")
	}
	if err := plain.Save(); err != nil {
		t.Fatal(err)
	}

	after, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(after.Recipients(), ",") != strings.Join(before.Recipients(), ",") {
		t.Error("recipients changed")
	}
	got := testutil.SopsDecrypt(t, p, testutil.KeyFile())
	want := "API_TOKEN=second-s3cr3t-test-value\nNEW_KEY=s3cr3t-test-value\n"
	if got != want {
		t.Errorf("sops decrypt after Save: wrong content")
	}
}

func TestSaveRefusesConcurrentChange(t *testing.T) {
	p, plain := decryptFixture(t)
	if _, err := plain.Set("NEW_KEY", "s3cr3t-test-value"); err != nil {
		t.Fatal(err)
	}
	other := []byte("changed by someone else\n")
	if err := os.WriteFile(p, other, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := plain.Save(); !errors.Is(err, ErrChanged) {
		t.Fatalf("want ErrChanged, got %v", err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(other) {
		t.Fatal("Save overwrote a concurrent change")
	}
}

func TestSaveKeepsFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes")
	}
	p := fixture(t)
	if err := os.Chmod(p, 0o640); err != nil {
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
	if err := plain.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode %o, want 640", info.Mode().Perm())
	}
}
