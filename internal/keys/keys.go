// Package keys finds, loads and generates the age private key sopsy uses.
package keys

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"filippo.io/age"
)

// EnvKeyFile names an explicit key file; it disables every other location.
const EnvKeyFile = "SOPS_AGE_KEY_FILE"

// Env is the slice of the process environment key lookup depends on. It is
// injectable so tests can fake every operating system's layout on any host.
type Env struct {
	GOOS   string
	Home   string
	Getenv func(string) string
}

// OSEnv returns the Env of the running process.
func OSEnv() Env {
	home, _ := os.UserHomeDir()
	return Env{GOOS: runtime.GOOS, Home: home, Getenv: os.Getenv}
}

// ConfigDir returns the directory SOPS itself uses for its default key file.
// It mirrors os.UserConfigDir, plus SOPS' rule that $XDG_CONFIG_HOME wins on
// macOS too.
func (e Env) ConfigDir() (string, error) {
	switch e.GOOS {
	case "windows":
		dir := e.Getenv("AppData")
		if dir == "" {
			return "", errors.New("%AppData% is not set")
		}
		return dir, nil
	case "darwin":
		if dir := e.Getenv("XDG_CONFIG_HOME"); dir != "" {
			return dir, nil
		}
		if e.Home == "" {
			return "", errors.New("home directory is unknown")
		}
		return filepath.Join(e.Home, "Library", "Application Support"), nil
	default:
		if dir := e.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
			return dir, nil
		}
		if e.Home == "" {
			return "", errors.New("home directory is unknown")
		}
		return filepath.Join(e.Home, ".config"), nil
	}
}

// DefaultPath is where `sopsy init` writes a new key: the location the sops
// CLI finds without any environment variable.
func (e Env) DefaultPath() (string, error) {
	dir, err := e.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sops", "age", "keys.txt"), nil
}

// Candidates returns the lookup order, without duplicates. When
// SOPS_AGE_KEY_FILE is set it is the only candidate.
func (e Env) Candidates() []string {
	if p := e.Getenv(EnvKeyFile); p != "" {
		return []string{p}
	}
	var out []string
	add := func(p string) {
		for _, seen := range out {
			if seen == p {
				return
			}
		}
		out = append(out, p)
	}
	if e.Home != "" {
		add(filepath.Join(e.Home, ".config", "sops", "age", "keys.txt"))
	}
	if p, err := e.DefaultPath(); err == nil {
		add(p)
	}
	return out
}

// NotFoundError reports that no key file exists in any searched location.
type NotFoundError struct {
	Searched []string
	// Explicit is true when SOPS_AGE_KEY_FILE named a missing file.
	Explicit bool
}

func (e *NotFoundError) Error() string {
	if e.Explicit {
		return fmt.Sprintf("%s points to %s, which does not exist", EnvKeyFile, e.Searched[0])
	}
	return "no age key found; searched " + strings.Join(e.Searched, ", ")
}

// Find returns the path of the first existing key file in lookup order.
func (e Env) Find() (string, error) {
	candidates := e.Candidates()
	for _, p := range candidates {
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return p, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("checking key file %s: %w", p, err)
		}
	}
	return "", &NotFoundError{Searched: candidates, Explicit: e.Getenv(EnvKeyFile) != ""}
}

// Key is a loaded age key file.
type Key struct {
	Path       string
	Identities []age.Identity
	// Public is the recipient of the first identity in the file.
	Public string
}

// Load finds and parses the key file.
func (e Env) Load() (*Key, error) {
	path, err := e.Find()
	if err != nil {
		return nil, err
	}
	return LoadFile(path)
}

// LoadFile parses an age key file. Errors never contain file content.
func LoadFile(path string) (*Key, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading key file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	ids, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("key file %s does not contain a valid age identity", path)
	}
	public, err := publicKey(ids[0])
	if err != nil {
		return nil, fmt.Errorf("key file %s: %w", path, err)
	}
	return &Key{Path: path, Identities: ids, Public: public}, nil
}

func publicKey(id age.Identity) (string, error) {
	switch id := id.(type) {
	case *age.X25519Identity:
		return id.Recipient().String(), nil
	case *age.HybridIdentity:
		return id.Recipient().String(), nil
	default:
		return "", fmt.Errorf("unsupported identity type %T", id)
	}
}

// Generate creates a new X25519 key file at path in the age-keygen format,
// readable only by the current user. It refuses to overwrite an existing file.
func Generate(path string) (*Key, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generating age key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating key directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("creating key file %s: %w", path, err)
	}
	w := bufio.NewWriter(f)
	public := id.Recipient().String()
	_, _ = fmt.Fprintf(w, "# created: %s\n", time.Now().UTC().Format(time.RFC3339))
	_, _ = fmt.Fprintf(w, "# public key: %s\n", public)
	_, _ = fmt.Fprintf(w, "%s\n", id)
	if err := errors.Join(w.Flush(), f.Close()); err != nil {
		return nil, fmt.Errorf("writing key file %s: %w", path, err)
	}
	return &Key{Path: path, Identities: []age.Identity{id}, Public: public}, nil
}

// IsRecipient reports whether any of the key's identities matches recipient.
func (k *Key) IsRecipient(recipient string) bool {
	for _, id := range k.Identities {
		if p, err := publicKey(id); err == nil && p == recipient {
			return true
		}
	}
	return false
}
