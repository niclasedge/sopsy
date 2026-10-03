package store

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/stores"
)

var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateName checks that name can be used as a key.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid key name %q: use letters, digits and _, not starting with a digit", name)
	}
	// SOPS reads every sops_* entry of a dotenv file as metadata.
	if strings.HasPrefix(name, stores.SopsMetadataKey+"_") {
		return fmt.Errorf("invalid key name %q: the prefix %s_ is reserved for SOPS metadata", name, stores.SopsMetadataKey)
	}
	return nil
}

// ValidateValue checks that value can be stored. The SOPS dotenv format has
// no unambiguous representation for line breaks.
func ValidateValue(value string) error {
	if value == "" {
		return fmt.Errorf("the value is empty")
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("the value contains a line break, which a dotenv file cannot store")
	}
	return nil
}

// Set adds or replaces a key and reports whether it already existed.
func (p *Plain) Set(name, value string) (existed bool, err error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	if err := ValidateValue(value); err != nil {
		return false, err
	}
	branch := p.file.tree.Branches[0]
	for i, item := range branch {
		if item.Key == name {
			branch[i].Value = value
			return true, nil
		}
	}
	p.file.tree.Branches[0] = append(branch, sops.TreeItem{Key: name, Value: value})
	return false, nil
}

// Unset removes a key and reports whether it existed.
func (p *Plain) Unset(name string) bool {
	branch := p.file.tree.Branches[0]
	for i, item := range branch {
		if item.Key == name {
			p.file.tree.Branches[0] = append(branch[:i], branch[i+1:]...)
			return true
		}
	}
	return false
}

// Save re-encrypts the file with its existing data key and recipients,
// recomputes the MAC and replaces the file atomically. It writes nothing and
// returns ErrChanged when the file changed on disk since Load. Plain must not
// be used after Save.
func (p *Plain) Save() error {
	data, err := encrypt(&p.file.tree, p.dataKey)
	if err != nil {
		return err
	}
	current, err := os.ReadFile(p.file.Path)
	if err != nil {
		return err
	}
	if sha256.Sum256(current) != p.file.hash {
		return fmt.Errorf("%s: %w", p.file.Path, ErrChanged)
	}
	// Keep the line endings the file had, so a CRLF checkout stays CRLF.
	if bytes.Contains(current, []byte("\r\n")) {
		data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	}
	return writeAtomic(p.file.Path, data, p.file.mode)
}
