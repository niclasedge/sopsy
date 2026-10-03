package sopsconfig

import (
	"errors"
	"flag"
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

var update = flag.Bool("update", false, "rewrite golden files")

// TestEditGolden checks that add and remove touch only the age list of the
// matching rule and keep comments and anchors.
func TestEditGolden(t *testing.T) {
	tests := []struct {
		golden string
		edit   func(r *Rule) error
	}{
		{"team.add.golden", func(r *Rule) error { return r.Add("age1dave") }},
		{"team.remove-alias.golden", func(r *Rule) error {
			removed, kept, err := r.Remove("age1bob")
			if !removed || !kept {
				t.Errorf("Remove(age1bob) = %v, %v; want removed with anchor kept", removed, kept)
			}
			return err
		}},
		{"team.remove-plain.golden", func(r *Rule) error {
			removed, kept, err := r.Remove("age1carol")
			if !removed || kept {
				t.Errorf("Remove(age1carol) = %v, %v; want removed, no alias", removed, kept)
			}
			return err
		}},
	}
	input, err := os.ReadFile(filepath.Join("testdata", "team.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.golden, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, FileName)
			write(t, path, string(input))
			c, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			rule, err := c.RuleFor(filepath.Join(dir, "secrets.env"))
			if err != nil {
				t.Fatal(err)
			}
			if err := tt.edit(rule); err != nil {
				t.Fatal(err)
			}
			if err := c.Save(); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", tt.golden)
			if *update {
				write(t, golden, string(got))
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestEditScalarAndMissingAgeField(t *testing.T) {
	tests := []struct{ config, want string }{
		{"creation_rules:\n  - age: age1a,age1b\n", "creation_rules:\n  - age: age1a,age1b,age1c\n"},
		{"creation_rules:\n  - pgp: ABC\n", "creation_rules:\n  - pgp: ABC\n    age:\n      - age1c\n"},
	}
	for _, tt := range tests {
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
		if err := rule.Add("age1c"); err != nil {
			t.Fatal(err)
		}
		if err := c.Save(); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != tt.want {
			t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
		}
	}
}

func TestEditAliasedListRefused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	write(t, path, "shared: &k\n  - age1a\ncreation_rules:\n  - age: *k\n")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := c.RuleFor(filepath.Join(dir, "secrets.env"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(rule.Recipients(), ","); got != "age1a" {
		t.Fatalf("Recipients() = %s", got)
	}
	if err := rule.Add("age1b"); err == nil {
		t.Fatal("Add edited a list defined through an alias")
	}
}
