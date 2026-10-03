// Package doctor checks each link from the age key to the decrypted secrets
// file and names the first broken one with a fix. The CLI prints the
// results; the web UI renders them on its setup page.
package doctor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/store"
)

// Status is the outcome of one check.
type Status string

// Check outcomes. Info is used for facts that can never fail.
const (
	OK      Status = "ok"
	Warn    Status = "warn"
	Fail    Status = "fail"
	Skipped Status = "skipped"
	Info    Status = "info"
)

// Result is what a check found.
type Result struct {
	Name   string
	Status Status
	Detail string
	// Fix is the command or step that resolves a warn or fail.
	Fix string
}

// Input is everything the checks look at.
type Input struct {
	Keys        keys.Env
	Dir         string
	SecretsFile string
	// LookPath finds external tools; nil means exec.LookPath.
	LookPath func(string) (string, error)
}

// state carries what earlier checks found to later ones.
type state struct {
	Input
	key        *keys.Key
	keyPath    string
	rule       *sopsconfig.Rule
	recipient  bool
	configPath string
	// file and fileErr cache loading the secrets file, which two checks use.
	file     *store.File
	fileErr  error
	fileRead bool
}

func (s *state) loadFile() (*store.File, error) {
	if !s.fileRead {
		s.file, s.fileErr = store.Load(s.SecretsFile)
		s.fileRead = true
	}
	return s.file, s.fileErr
}

type check struct {
	name string
	run  func(*state) Result
}

var checks = []check{
	{"age key", checkKey},
	{"key permissions", checkPermissions},
	{".sops.yaml", checkConfig},
	{"recipient", checkRecipient},
	{"secrets file", checkFile},
	{"decrypt", checkDecrypt},
	{"sops/age CLI", checkTools},
}

// Run runs every check in order.
func Run(in Input) []Result {
	if in.LookPath == nil {
		in.LookPath = exec.LookPath
	}
	s := &state{Input: in}
	out := make([]Result, 0, len(checks))
	for _, c := range checks {
		r := c.run(s)
		r.Name = c.name
		out = append(out, r)
	}
	return out
}

// Failed reports whether any result is a failure.
func Failed(results []Result) bool {
	return slices.ContainsFunc(results, func(r Result) bool { return r.Status == Fail })
}

func skipped(why string) Result { return Result{Status: Skipped, Detail: why} }

func checkKey(s *state) Result {
	path, err := s.Keys.Find()
	var nf *keys.NotFoundError
	switch {
	case errors.As(err, &nf) && nf.Explicit:
		return Result{Status: Fail, Detail: nf.Error(), Fix: "create that file, or unset " + keys.EnvKeyFile}
	case errors.As(err, &nf):
		return Result{Status: Fail, Detail: "not found in " + strings.Join(nf.Searched, ", "), Fix: "sopsy init"}
	case err != nil:
		return Result{Status: Fail, Detail: err.Error()}
	}
	s.keyPath = path
	k, err := keys.LoadFile(path)
	if err != nil {
		return Result{Status: Fail, Detail: err.Error(), Fix: "restore the key file from your backup"}
	}
	s.key = k
	return Result{Status: OK, Detail: path}
}

func checkPermissions(s *state) Result {
	if s.keyPath == "" {
		return skipped("no key file")
	}
	if runtime.GOOS == "windows" {
		return Result{Status: OK, Detail: "protected by the user profile's access rules"}
	}
	info, err := os.Stat(s.keyPath)
	if err != nil {
		return Result{Status: Fail, Detail: err.Error()}
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return Result{
			Status: Warn,
			Detail: fmt.Sprintf("mode %04o: other users on this machine can read the key", mode),
			Fix:    "chmod 600 " + s.keyPath,
		}
	}
	return Result{Status: OK, Detail: "readable only by you"}
}

func checkConfig(s *state) Result {
	path, err := sopsconfig.Find(s.Dir)
	if errors.Is(err, sopsconfig.ErrNotFound) {
		return Result{Status: Fail, Detail: "not found in this directory or any parent", Fix: "sopsy init"}
	}
	if err != nil {
		return Result{Status: Fail, Detail: err.Error()}
	}
	s.configPath = path
	c, err := sopsconfig.Load(path)
	if err != nil {
		return Result{Status: Fail, Detail: err.Error(), Fix: "fix the YAML syntax in " + path}
	}
	rule, err := c.RuleFor(s.SecretsFile)
	if err != nil {
		return Result{Status: Fail, Detail: err.Error(), Fix: "add a creation rule for the secrets file to " + path}
	}
	s.rule = rule
	return Result{Status: OK, Detail: path}
}

func checkFile(s *state) Result {
	_, err := s.loadFile()
	if errors.Is(err, fs.ErrNotExist) {
		return Result{Status: Fail, Detail: s.SecretsFile + " does not exist", Fix: "sopsy init"}
	}
	if err != nil {
		return Result{Status: Fail, Detail: err.Error()}
	}
	return Result{Status: OK, Detail: s.SecretsFile}
}

func checkRecipient(s *state) Result {
	if s.key == nil {
		return skipped("no key")
	}
	// The file's own recipient list decides; .sops.yaml only matters for
	// files that do not exist yet.
	var recipients []string
	var source string
	f, _ := s.loadFile()
	switch {
	case f != nil:
		recipients, source = f.Recipients(), s.SecretsFile
	case s.rule != nil:
		recipients, source = s.rule.Recipients(), s.configPath
	default:
		return skipped("no secrets file and no .sops.yaml rule")
	}
	if !slices.ContainsFunc(recipients, s.key.IsRecipient) {
		return Result{
			Status: Fail,
			Detail: fmt.Sprintf("your key %s is not a recipient of %s", s.key.Public, source),
			Fix:    "send the output of `sopsy pubkey` to someone who can decrypt; they run `sopsy recipients add " + s.key.Public + "`",
		}
	}
	s.recipient = true
	return Result{Status: OK, Detail: "your key can decrypt " + source}
}

func checkDecrypt(s *state) Result {
	if s.key == nil || s.file == nil || !s.recipient {
		return skipped("an earlier check failed")
	}
	plain, err := s.file.Decrypt(s.key.Identities, s.key.Public)
	var tam *store.TamperedError
	if errors.As(err, &tam) {
		return Result{Status: Fail, Detail: tam.Error(), Fix: "restore the file from version control"}
	}
	if err != nil {
		return Result{Status: Fail, Detail: err.Error()}
	}
	n := len(plain.Entries())
	unit := "keys"
	if n == 1 {
		unit = "key"
	}
	return Result{Status: OK, Detail: fmt.Sprintf("%d %s", n, unit)}
}

func checkTools(s *state) Result {
	var parts []string
	for _, tool := range []string{"sops", "age"} {
		path, err := s.LookPath(tool)
		if err != nil {
			parts = append(parts, tool+" not installed")
			continue
		}
		parts = append(parts, tool+" "+toolVersion(path))
	}
	return Result{Status: Info, Detail: strings.Join(parts, ", ") + " (sopsy does not need them)"}
}

func toolVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	// sops would otherwise ask GitHub for the latest release.
	cmd.Env = append(os.Environ(), "SOPS_DISABLE_VERSION_CHECK=1")
	out, err := cmd.Output()
	if err != nil {
		return "at " + path
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimPrefix(strings.TrimPrefix(line, "sops "), "age ") + " at " + path
}
