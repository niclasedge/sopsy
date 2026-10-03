// Package cli implements sopsy's command line: dispatch, flags, exit codes and
// error reporting shared by every command.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/niclasedge/sopsy/internal/keys"
)

// Exit codes follow env(1): 125 when sopsy itself fails; commands that run a
// child use 126/127 and otherwise pass the child's code through.
const (
	ExitOK   = 0
	ExitFail = 125
)

// EnvFile names the secrets file when --file is not given.
const (
	EnvFile     = "SOPSY_FILE"
	DefaultFile = "secrets.env"
)

// Env is everything a command may touch outside its arguments. Tests build
// one around temp directories; Main builds it from the real process.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Dir is the working directory; relative paths resolve against it.
	Dir    string
	Getenv func(string) string
	// Environ is the environment handed to child processes.
	Environ func() []string
	Keys    keys.Env
	// ReadSecret reads one line from the terminal without echo. It is nil
	// when stdin is not a terminal; values are then read from Stdin.
	ReadSecret func(prompt string) (string, error)
}

// Main runs sopsy for the current process and returns its exit code.
func Main() int {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sopsy: cannot determine working directory: %v\n", err)
		return ExitFail
	}
	return Run(Env{
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Dir:        dir,
		Getenv:     os.Getenv,
		Environ:    os.Environ,
		Keys:       keys.OSEnv(),
		ReadSecret: terminalReader(),
	}, os.Args[1:])
}

// terminalReader returns a hidden-input reader when stdin is a terminal.
func terminalReader() func(string) (string, error) {
	fd := int(os.Stdin.Fd()) //nolint:gosec // file descriptors fit in an int
	if !term.IsTerminal(fd) {
		return nil
	}
	return func(prompt string) (string, error) {
		_, _ = fmt.Fprint(os.Stderr, prompt)
		b, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(os.Stderr)
		return string(b), err
	}
}

type command struct {
	summary string
	run     func(env Env, args []string) error
}

var commands = map[string]command{
	"init":       {"create age key, .sops.yaml and secrets file (only what is missing)", runInit},
	"set":        {"set a key; value from the hidden prompt or stdin, never argv", runSet},
	"unset":      {"remove a key", runUnset},
	"keys":       {"list key names (no key needed, values never shown)", runKeys},
	"run":        {"run a command with the secrets in its environment: sopsy run -- CMD", runRun},
	"pubkey":     {"print this machine's age public key", runPubkey},
	"recipients": {"list recipients; recipients add|remove AGE_PUBLIC_KEY", runRecipients},
	"doctor":     {"check key, permissions, config, recipients and decryption", runDoctor},
}

// Run executes one sopsy invocation and returns its exit code.
func Run(env Env, args []string) int {
	if len(args) == 0 {
		usage(env.Stderr)
		return ExitFail
	}
	name := args[0]
	if name == "help" || name == "-h" || name == "--help" {
		usage(env.Stdout)
		return ExitOK
	}
	cmd, ok := commands[name]
	if !ok {
		return report(env, &Error{
			Msg:  fmt.Sprintf("unknown command %q", name),
			Hint: []string{"run `sopsy help` for the list of commands"},
		})
	}
	return report(env, cmd.run(env, args[1:]))
}

func usage(w io.Writer) {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("sopsy - set and use secrets from a SOPS + age encrypted dotenv file\n\nUsage:\n")
	for _, name := range names {
		fmt.Fprintf(&b, "  sopsy %-10s %s\n", name, commands[name].summary)
	}
	b.WriteString("\nThe secrets file is --file, else $SOPSY_FILE, else secrets.env.\n")
	_, _ = io.WriteString(w, b.String())
}

// report prints err and returns the matching exit code.
func report(env Env, err error) int {
	if err == nil {
		return ExitOK
	}
	var code *exitCode
	if errors.As(err, &code) {
		return code.code
	}
	e := explain(err)
	var b strings.Builder
	fmt.Fprintf(&b, "sopsy: %s\n", e.Msg)
	for _, h := range e.Hint {
		fmt.Fprintf(&b, "  %s\n", h)
	}
	_, _ = io.WriteString(env.Stderr, b.String())
	if e.Code != 0 {
		return e.Code
	}
	return ExitFail
}

// exitCode ends a command with a specific code without printing anything,
// for commands whose output already explains the outcome.
type exitCode struct{ code int }

func (e *exitCode) Error() string { return fmt.Sprintf("exit %d", e.code) }

// flags returns a flag set for a subcommand that reports parse errors as
// usage errors instead of exiting.
func flags(env Env, name string) *flag.FlagSet {
	fs := flag.NewFlagSet("sopsy "+name, flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	return fs
}

// parse parses args and turns flag errors into an *Error.
func parse(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return &exitCode{ExitOK}
		}
		return &Error{Msg: err.Error(), Hint: []string{"run `" + fs.Name() + " -h` for usage"}}
	}
	return nil
}

// fileFlag registers --file on fs.
func fileFlag(fs *flag.FlagSet) *string {
	return fs.String("file", "", "secrets file (default $SOPSY_FILE, else secrets.env)")
}

// secretsFile resolves the secrets file: flag, else $SOPSY_FILE, else
// secrets.env, relative to the working directory.
func (env Env) secretsFile(flagValue string) string {
	p := flagValue
	if p == "" {
		p = env.Getenv(EnvFile)
	}
	if p == "" {
		p = DefaultFile
	}
	return env.abs(p)
}

func (env Env) abs(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(env.Dir, p)
}
