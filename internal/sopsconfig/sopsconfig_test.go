package sopsconfig

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindInParentDirectory(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, FileName)
	write(t, want, "creation_rules: []\n")
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Find() = %s, want %s", got, want)
	}
}

func TestFindNotFound(t *testing.T) {
	// A temp dir has no .sops.yaml in it; a parent might on a developer
	// machine, so only assert the error when nothing was found.
	got, err := Find(t.TempDir())
	if err == nil {
		t.Skipf("a parent of the temp dir has %s: %s", FileName, got)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPathRegex(t *testing.T) {
	re := regexp.MustCompile(PathRegex("/x/secrets.env"))
	for _, s := range []string{"secrets.env", "app/secrets.env", `app\secrets.env`} {
		if !re.MatchString(s) {
			t.Errorf("%s does not match", s)
		}
	}
	for _, s := range []string{"secretsXenv", "my-secrets.env", "secrets.env.bak"} {
		if re.MatchString(s) {
			t.Errorf("%s matches", s)
		}
	}
}

func TestCreateAndRuleFor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	secrets := filepath.Join(dir, "secrets.env")
	if err := Create(path, secrets, "age1own"); err != nil {
		t.Fatal(err)
	}
	if err := Create(path, secrets, "age1own"); err == nil {
		t.Fatal("Create overwrote an existing file")
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := c.RuleFor(secrets)
	if err != nil {
		t.Fatal(err)
	}
	if got := rule.Recipients(); len(got) != 1 || got[0] != "age1own" {
		t.Fatalf("Recipients() = %v", got)
	}
	if _, err := c.RuleFor(filepath.Join(dir, "other.env")); err == nil {
		t.Fatal("rule for secrets.env matched other.env")
	}
}

func TestRuleForForms(t *testing.T) {
	tests := []struct {
		name, config string
		want         string
	}{
		{"comma scalar", "creation_rules:\n  - age: age1a, age1b\n", "age1a,age1b"},
		{"sequence", "creation_rules:\n  - age:\n      - age1a\n      - age1b\n", "age1a,age1b"},
		{"anchors", "keys:\n  - &a age1a\ncreation_rules:\n  - age:\n      - *a\n      - age1b\n", "age1a,age1b"},
		{"one key group", "creation_rules:\n  - key_groups:\n      - age:\n          - age1a\n", "age1a"},
		{"first match wins", "creation_rules:\n  - path_regex: other\n    age: age1x\n  - path_regex: secrets\n    age: age1a\n  - age: age1y\n", "age1a"},
		{"no age field", "creation_rules:\n  - pgp: ABC\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, FileName)
			write(t, path, tt.config)
			c, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			rule, err := c.RuleFor(filepath.Join(dir, "secrets.env"))
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(rule.Recipients(), ","); got != tt.want {
				t.Fatalf("Recipients() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRuleForPathRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	// ^ anchors at the config directory, as in SOPS.
	write(t, path, "creation_rules:\n  - path_regex: ^secrets\\.env$\n    age: age1a\n")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.RuleFor(filepath.Join(dir, "secrets.env")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RuleFor(filepath.Join(dir, "sub", "secrets.env")); err == nil {
		t.Fatal("anchored rule matched a file in a subdirectory")
	}
}

func TestSeveralKeyGroupsUnsupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	write(t, path, "creation_rules:\n  - key_groups:\n      - age: [age1a]\n      - age: [age1b]\n")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.RuleFor(filepath.Join(dir, "secrets.env")); err == nil {
		t.Fatal("want error for several key groups")
	}
}
