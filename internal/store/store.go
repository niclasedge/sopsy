// Package store reads and writes SOPS-encrypted dotenv files through the SOPS
// library, so files stay format-identical to what the sops CLI produces.
package store

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"filippo.io/age"
	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/stores"
	"github.com/getsops/sops/v3/stores/dotenv"
	"github.com/getsops/sops/v3/version"

	"github.com/niclasedge/sopsy/internal/atomicfile"
)

// NotRecipientError means the loaded key cannot decrypt the file's data key.
type NotRecipientError struct {
	Path   string
	Public string
}

func (e *NotRecipientError) Error() string {
	return fmt.Sprintf("your age key %s is not a recipient of %s", e.Public, e.Path)
}

// TamperedError means the file's MAC or a ciphertext does not verify.
type TamperedError struct{ Path string }

func (e *TamperedError) Error() string {
	return fmt.Sprintf("%s failed its integrity check: it was modified outside SOPS", e.Path)
}

// ParseError points at a malformed line without repeating its content.
type ParseError struct {
	Path string
	Line int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s line %d is not a KEY=VALUE line", e.Path, e.Line)
}

// ErrChanged means the file was modified on disk after it was read.
var ErrChanged = errors.New("file changed on disk")

// ErrNotEncrypted means the file has no SOPS metadata.
var ErrNotEncrypted = errors.New("file is not encrypted with SOPS")

var dotenvStore = dotenv.NewStore(&config.DotenvStoreConfig{})

// File is an encrypted secrets file as read from disk. Key names and
// recipients are available without decrypting.
type File struct {
	Path string
	tree sops.Tree
	// hash and mode of the file as read, for change detection on write.
	hash [sha256.Size]byte
	mode fs.FileMode
}

// Load reads and parses an encrypted dotenv file without decrypting it.
func Load(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(raw)
	// A Windows checkout with core.autocrlf turns LF into CRLF. The MAC
	// covers values, not line endings, so CRLF is read as LF.
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	// The SOPS dotenv parser quotes the offending line in its errors, so
	// malformed lines are reported here first, by number only.
	for i, line := range bytes.Split(raw, []byte("\n")) {
		if len(line) > 0 && line[0] != '#' && !bytes.Contains(line, []byte("=")) {
			return nil, &ParseError{Path: path, Line: i + 1}
		}
	}
	tree, err := dotenvStore.LoadEncryptedFile(raw)
	if errors.Is(err, sops.MetadataNotFound) {
		return nil, fmt.Errorf("%s: %w", path, ErrNotEncrypted)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: cannot read SOPS metadata: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	tree.FilePath = abs
	return &File{Path: path, tree: tree, hash: hash, mode: info.Mode().Perm()}, nil
}

// Names returns the key names in file order.
func (f *File) Names() []string {
	var names []string
	for _, item := range f.tree.Branches[0] {
		if name, ok := item.Key.(string); ok {
			names = append(names, name)
		}
	}
	return names
}

// Recipients returns the age recipients in the file's metadata.
func (f *File) Recipients() []string {
	var out []string
	for _, group := range f.tree.Metadata.KeyGroups {
		for _, k := range group {
			if ak, ok := k.(*sopsage.MasterKey); ok {
				out = append(out, ak.Recipient)
			}
		}
	}
	return out
}

// Plain is a decrypted secrets file. It only lives in memory.
type Plain struct {
	file    *File
	dataKey []byte
}

// Entry is one decrypted key/value pair.
type Entry struct {
	Name  string
	Value string
}

// Decrypt decrypts the file with the given identities. public is the
// caller's public key, used only in the NotRecipientError message.
func (f *File) Decrypt(ids []age.Identity, public string) (*Plain, error) {
	dataKey, err := f.tree.Metadata.GetDataKeyWithKeyServices(clients(ids), nil)
	if err != nil {
		return nil, &NotRecipientError{Path: f.Path, Public: public}
	}
	cipher := aes.NewCipher()
	mac, err := f.tree.Decrypt(dataKey, cipher)
	if err != nil {
		return nil, &TamperedError{Path: f.Path}
	}
	stored, err := cipher.Decrypt(f.tree.Metadata.MessageAuthenticationCode, dataKey,
		f.tree.Metadata.LastModified.Format(time.RFC3339))
	if err != nil || stored != mac {
		return nil, &TamperedError{Path: f.Path}
	}
	return &Plain{file: f, dataKey: dataKey}, nil
}

// Entries returns the decrypted key/value pairs in file order.
func (p *Plain) Entries() []Entry {
	var out []Entry
	for _, item := range p.file.tree.Branches[0] {
		name, ok := item.Key.(string)
		if !ok {
			continue
		}
		value, ok := item.Value.(string)
		if !ok {
			value = stores.ValToString(item.Value)
		}
		out = append(out, Entry{Name: name, Value: value})
	}
	return out
}

// Create writes a new, empty encrypted secrets file at path using the
// recipients and settings of the given creation rule. It never overwrites.
func Create(path string, rule *config.Config) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	tree := sops.Tree{
		Branches: sops.TreeBranches{sops.TreeBranch{}},
		Metadata: sops.Metadata{
			KeyGroups:               rule.KeyGroups,
			ShamirThreshold:         rule.ShamirThreshold,
			UnencryptedSuffix:       rule.UnencryptedSuffix,
			EncryptedSuffix:         rule.EncryptedSuffix,
			UnencryptedRegex:        rule.UnencryptedRegex,
			EncryptedRegex:          rule.EncryptedRegex,
			UnencryptedCommentRegex: rule.UnencryptedCommentRegex,
			EncryptedCommentRegex:   rule.EncryptedCommentRegex,
			MACOnlyEncrypted:        rule.MACOnlyEncrypted,
			Version:                 version.Version,
		},
		FilePath: abs,
	}
	if len(rule.KeyGroups) == 0 {
		return errors.New("the creation rule has no recipients")
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices(clients(nil))
	if len(errs) > 0 {
		return fmt.Errorf("sopsy can only encrypt to age recipients: %w", errors.Join(errs...))
	}
	data, err := encrypt(&tree, dataKey)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 0o644)
}

// encrypt encrypts the tree in place, recomputes the MAC and returns the
// file content.
func encrypt(tree *sops.Tree, dataKey []byte) ([]byte, error) {
	cipher := aes.NewCipher()
	mac, err := tree.Encrypt(dataKey, cipher)
	if err != nil {
		return nil, fmt.Errorf("encrypting: %w", err)
	}
	tree.Metadata.LastModified = time.Now().UTC()
	tree.Metadata.MessageAuthenticationCode, err = cipher.Encrypt(mac, dataKey,
		tree.Metadata.LastModified.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("encrypting MAC: %w", err)
	}
	return dotenvStore.EmitEncryptedFile(*tree)
}
