package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/niclasedge/sopsy/internal/store"
)

// decrypt loads the secrets file and decrypts it with the found key.
func (env Env) decrypt(file string) (*store.Plain, error) {
	f, err := store.Load(file)
	if err != nil {
		return nil, err
	}
	k, err := env.Keys.Load()
	if err != nil {
		return nil, err
	}
	return f.Decrypt(k.Identities, k.Public)
}

func runSet(env Env, args []string) error {
	fs := flags(env, "set")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	switch fs.NArg() {
	case 0:
		return &Error{Msg: "missing key name", Hint: []string{"usage: sopsy set [--file F] KEY  (value from the prompt or stdin)"}}
	case 1:
	default:
		return &Error{
			Msg: "set takes only the key name; values are read from stdin or the hidden prompt, never from arguments",
			Hint: []string{
				"type the value at the hidden prompt: sopsy set " + fs.Arg(0),
				"or pipe it in: printf '%s' \"$VALUE\" | sopsy set " + fs.Arg(0),
			},
		}
	}
	name := fs.Arg(0)
	if err := store.ValidateName(name); err != nil {
		return err
	}
	plain, err := env.decrypt(env.secretsFile(*file))
	if err != nil {
		return err
	}
	value, err := env.readValue(name)
	if err != nil {
		return err
	}
	if err := store.ValidateValue(value); err != nil {
		return &Error{Msg: err.Error() + "; nothing was written"}
	}
	existed, err := plain.Set(name, value)
	if err != nil {
		return err
	}
	if err := plain.Save(); err != nil {
		return err
	}
	if existed {
		_, _ = fmt.Fprintf(env.Stdout, "replaced %s\n", name)
	} else {
		_, _ = fmt.Fprintf(env.Stdout, "added %s\n", name)
	}
	return nil
}

// readValue reads a value from the hidden prompt when stdin is a terminal,
// else from stdin with at most one trailing newline removed.
func (env Env) readValue(name string) (string, error) {
	if env.ReadSecret != nil {
		first, err := env.ReadSecret("Value for " + name + ": ")
		if err != nil {
			return "", fmt.Errorf("reading value: %w", err)
		}
		second, err := env.ReadSecret("Repeat value: ")
		if err != nil {
			return "", fmt.Errorf("reading value: %w", err)
		}
		if first != second {
			return "", &Error{Msg: "the two entries do not match; nothing was written"}
		}
		return first, nil
	}
	data, err := io.ReadAll(env.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading value from stdin: %w", err)
	}
	value := string(data)
	if v, ok := strings.CutSuffix(value, "\r\n"); ok {
		return v, nil
	}
	value, _ = strings.CutSuffix(value, "\n")
	return value, nil
}

func runUnset(env Env, args []string) error {
	fs := flags(env, "unset")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return &Error{Msg: "unset takes exactly one key name", Hint: []string{"usage: sopsy unset [--file F] KEY"}}
	}
	name := fs.Arg(0)
	if err := store.ValidateName(name); err != nil {
		return err
	}
	plain, err := env.decrypt(env.secretsFile(*file))
	if err != nil {
		return err
	}
	if !plain.Unset(name) {
		_, _ = fmt.Fprintf(env.Stdout, "%s not found, nothing removed\n", name)
		return nil
	}
	if err := plain.Save(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(env.Stdout, "removed %s\n", name)
	return nil
}

func runKeys(env Env, args []string) error {
	fs := flags(env, "keys")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return &Error{Msg: "keys takes no arguments", Hint: []string{"usage: sopsy keys [--file F]"}}
	}
	f, err := store.Load(env.secretsFile(*file))
	if err != nil {
		return err
	}
	names := f.Names()
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString(n + "\n")
	}
	_, _ = io.WriteString(env.Stdout, b.String())
	return nil
}
