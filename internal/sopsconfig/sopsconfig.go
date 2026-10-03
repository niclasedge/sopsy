// Package sopsconfig finds, reads and edits .sops.yaml while keeping its
// comments, anchors and ordering intact.
package sopsconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/niclasedge/sopsy/internal/atomicfile"
)

// FileName is the only config file name sopsy recognises, as in SOPS.
const FileName = ".sops.yaml"

// ErrNotFound means no .sops.yaml exists in the directory or any parent.
var ErrNotFound = errors.New(FileName + " not found")

// Find searches dir and its parents for .sops.yaml.
func Find(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, FileName)
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return p, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("checking %s: %w", p, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// PathRegex returns a creation-rule regex matching secretsFile by base name in
// any directory, with either path separator.
func PathRegex(secretsFile string) string {
	return `(^|[/\\])` + regexp.QuoteMeta(filepath.Base(secretsFile)) + `$`
}

// Create writes a new .sops.yaml at path with one creation rule for
// secretsFile and recipient as its only age recipient. It never overwrites.
func Create(path, secretsFile, recipient string) error {
	content := fmt.Sprintf("creation_rules:\n  - path_regex: '%s'\n    age:\n      - %s\n",
		PathRegex(secretsFile), recipient)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	_, err = f.WriteString(content)
	if err := errors.Join(err, f.Close()); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Config is a parsed .sops.yaml.
type Config struct {
	Path string
	doc  yaml.Node
	// crlf records CRLF line endings so Save writes them back.
	crlf bool
}

// Load parses the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	// yaml.v3 mangles comments in CRLF input (e.g. a Windows checkout), so
	// parse LF and restore the line endings on Save.
	c := &Config{Path: path, crlf: bytes.Contains(data, []byte("\r\n"))}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if err := yaml.Unmarshal(data, &c.doc); err != nil {
		return nil, fmt.Errorf("%s is not valid YAML: %w", path, err)
	}
	return c, nil
}

// Rule is the creation rule that applies to one secrets file.
type Rule struct {
	// age is the node holding the age recipients: a sequence or a
	// comma-separated scalar. nil when the rule has no age field.
	age *yaml.Node
	// parent is the mapping that holds (or will hold) the age field.
	parent *yaml.Node
}

// RuleFor returns the first creation rule whose path_regex matches
// secretsFile, using the same matching as SOPS: the path relative to the
// config file's directory, and a rule without path_regex matches everything.
func (c *Config) RuleFor(secretsFile string) (*Rule, error) {
	rules := mapValue(root(&c.doc), "creation_rules")
	if rules == nil || rules.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s has no creation_rules", c.Path)
	}
	abs, err := filepath.Abs(secretsFile)
	if err != nil {
		return nil, err
	}
	configDir, err := filepath.Abs(filepath.Dir(c.Path))
	if err != nil {
		return nil, err
	}
	rel := strings.TrimPrefix(abs, configDir+string(filepath.Separator))
	for _, r := range rules.Content {
		r = resolve(r)
		if r.Kind != yaml.MappingNode {
			continue
		}
		if re := mapValue(r, "path_regex"); re != nil && re.Value != "" {
			ok, err := regexp.MatchString(resolve(re).Value, rel)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid path_regex %q: %w", c.Path, re.Value, err)
			}
			if !ok {
				continue
			}
		}
		return ruleFrom(c.Path, r)
	}
	return nil, fmt.Errorf("no creation rule in %s matches %s", c.Path, rel)
}

func ruleFrom(path string, r *yaml.Node) (*Rule, error) {
	if age := mapValue(r, "age"); age != nil {
		return &Rule{age: age, parent: r}, nil
	}
	groups := mapValue(r, "key_groups")
	if groups == nil {
		return &Rule{parent: r}, nil
	}
	groups = resolve(groups)
	if groups.Kind != yaml.SequenceNode || len(groups.Content) != 1 {
		return nil, fmt.Errorf("%s: creation rules with several key_groups are not supported; edit the file with the sops CLI instead", path)
	}
	group := resolve(groups.Content[0])
	return &Rule{age: mapValue(group, "age"), parent: group}, nil
}

// Recipients returns the age recipients of the rule, with aliases resolved.
func (r *Rule) Recipients() []string {
	if r.age == nil {
		return nil
	}
	n := resolve(r.age)
	if n.Kind == yaml.ScalarNode {
		return splitScalar(n.Value)
	}
	var out []string
	for _, item := range n.Content {
		out = append(out, splitScalar(resolve(item).Value)...)
	}
	return out
}

func splitScalar(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Add appends recipient to the rule's age recipients.
func (r *Rule) Add(recipient string) error {
	switch {
	case r.age == nil:
		r.age = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		r.parent.Content = append(r.parent.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "age"}, r.age)
		fallthrough
	case r.age.Kind == yaml.SequenceNode:
		r.age.Content = append(r.age.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: recipient})
	case r.age.Kind == yaml.ScalarNode:
		r.age.Value = strings.Join(append(splitScalar(r.age.Value), recipient), ",")
	default:
		return errors.New("the age recipients are defined through an alias; edit " + FileName + " by hand")
	}
	return nil
}

// Remove deletes recipient from the rule's age recipients. aliasKept reports
// that the entry was an alias whose anchor definition stays in the file.
func (r *Rule) Remove(recipient string) (removed, aliasKept bool, err error) {
	if r.age == nil {
		return false, false, nil
	}
	switch r.age.Kind {
	case yaml.ScalarNode:
		parts := splitScalar(r.age.Value)
		kept := slices.DeleteFunc(slices.Clone(parts), func(p string) bool { return p == recipient })
		r.age.Value = strings.Join(kept, ",")
		return len(kept) != len(parts), false, nil
	case yaml.SequenceNode:
		for i, item := range r.age.Content {
			if resolve(item).Value == recipient {
				r.age.Content = slices.Delete(r.age.Content, i, i+1)
				return true, item.Kind == yaml.AliasNode, nil
			}
		}
		return false, false, nil
	default:
		return false, false, errors.New("the age recipients are defined through an alias; edit " + FileName + " by hand")
	}
}

// Save writes the config back, preserving comments and anchors.
func (c *Config) Save() error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&c.doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	data := buf.Bytes()
	if c.crlf {
		data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	}
	info, err := os.Stat(c.Path)
	if err != nil {
		return err
	}
	return atomicfile.Write(c.Path, data, info.Mode().Perm())
}

func root(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		return doc.Content[0]
	}
	return doc
}

func resolve(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	m = resolve(m)
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}
