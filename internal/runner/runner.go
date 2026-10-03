// Package runner starts a command with extra environment variables and passes
// its exit code and interrupts through, like env(1) and `sops exec-env`.
package runner

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"strings"
)

// Exit codes for a command that could not be started, as in env(1).
const (
	ExitNotExecutable = 126
	ExitNotFound      = 127
)

// MergeEnv returns base with every NAME=VALUE entry of add set, replacing
// variables of the same name. foldCase compares names case-insensitively,
// as Windows does.
func MergeEnv(base, add []string, foldCase bool) []string {
	key := func(kv string) string {
		name, _, _ := strings.Cut(kv, "=")
		if foldCase {
			return strings.ToUpper(name)
		}
		return name
	}
	replaced := make(map[string]bool, len(add))
	for _, kv := range add {
		replaced[key(kv)] = true
	}
	out := make([]string, 0, len(base)+len(add))
	for _, kv := range base {
		// Windows keeps per-drive variables named like "=C:"; they have an
		// empty name here and are never replaced.
		if k := key(kv); k == "" || !replaced[k] {
			out = append(out, kv)
		}
	}
	return append(out, add...)
}

// StartError means the command could not be started; Code is 126 or 127.
type StartError struct {
	Code int
	Err  error
}

func (e *StartError) Error() string { return e.Err.Error() }
func (e *StartError) Unwrap() error { return e.Err }

// Command describes the child process.
type Command struct {
	Args   []string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Run starts the command, forwards interrupts to it, waits for it and
// returns its exit code. A child killed by a signal yields 128+signal.
func Run(c Command) (int, error) {
	path, err := exec.LookPath(c.Args[0])
	if err != nil {
		return 0, startError(c.Args[0], err)
	}
	cmd := exec.Command(path, c.Args[1:]...)
	cmd.Env = c.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = c.Stdin, c.Stdout, c.Stderr

	stop := forwardSignals(cmd)
	defer stop()
	if err := cmd.Start(); err != nil {
		return 0, &StartError{Code: ExitNotExecutable, Err: fmt.Errorf("cannot run %s: %w", c.Args[0], err)}
	}
	err = cmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return 0, fmt.Errorf("waiting for %s: %w", c.Args[0], err)
	}
	return exitCode(cmd.ProcessState), nil
}

func startError(name string, err error) error {
	switch {
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		return &StartError{Code: ExitNotFound, Err: fmt.Errorf("command not found: %s", name)}
	case errors.Is(err, exec.ErrDot):
		return &StartError{Code: ExitNotFound, Err: fmt.Errorf("%s resolves to the current directory via PATH; run it as ./%s", name, name)}
	default:
		return &StartError{Code: ExitNotExecutable, Err: fmt.Errorf("cannot run %s: %w", name, err)}
	}
}
