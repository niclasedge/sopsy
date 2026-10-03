package cli

import (
	"fmt"
	"strings"

	"github.com/niclasedge/sopsy/internal/recipients"
	"github.com/niclasedge/sopsy/internal/store"
)

func runPubkey(env Env, args []string) error {
	fs := flags(env, "pubkey")
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return &Error{Msg: "pubkey takes no arguments", Hint: []string{"usage: sopsy pubkey"}}
	}
	k, err := env.Keys.Load()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(env.Stdout, k.Public)
	return err
}

func runRecipients(env Env, args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return recipientsAdd(env, args[1:])
		case "remove":
			return recipientsRemove(env, args[1:])
		}
	}
	fs := flags(env, "recipients")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return &Error{Msg: fmt.Sprintf("unknown recipients subcommand %q", fs.Arg(0)), Hint: []string{
			"usage: sopsy recipients [--file F] | recipients add|remove [--file F] AGE_PUBLIC_KEY",
		}}
	}
	f, err := store.Load(env.secretsFile(*file))
	if err != nil {
		return err
	}
	// Listing needs no private key; without one there is just no marker.
	k, _ := env.Keys.Load()
	var b strings.Builder
	for _, r := range f.Recipients() {
		b.WriteString(r)
		if k != nil && k.IsRecipient(r) {
			b.WriteString("  (this machine)")
		}
		b.WriteString("\n")
	}
	_, err = fmt.Fprint(env.Stdout, b.String())
	return err
}

// recipientArgs parses `recipients add|remove [--file F] AGE_PUBLIC_KEY`.
func recipientArgs(env Env, name string, args []string) (file, recipient string, err error) {
	fs := flags(env, "recipients "+name)
	f := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return "", "", err
	}
	if fs.NArg() != 1 {
		return "", "", &Error{Msg: name + " takes exactly one age public key", Hint: []string{
			"usage: sopsy recipients " + name + " [--file F] AGE_PUBLIC_KEY",
		}}
	}
	return env.secretsFile(*f), fs.Arg(0), nil
}

func recipientsAdd(env Env, args []string) error {
	file, recipient, err := recipientArgs(env, "add", args)
	if err != nil {
		return err
	}
	o, err := recipients.Add(env.Keys, env.Dir, file, recipient)
	if err != nil {
		return err
	}
	if !o.Changed {
		_, err := fmt.Fprintf(env.Stdout, "%s is already a recipient; nothing changed\n", o.Recipient)
		return err
	}
	_, err = fmt.Fprintf(env.Stdout, "added %s\n  %s and %s updated\n", o.Recipient, o.File, o.Config)
	return err
}

func recipientsRemove(env Env, args []string) error {
	file, recipient, err := recipientArgs(env, "remove", args)
	if err != nil {
		return err
	}
	o, err := recipients.Remove(env.Keys, env.Dir, file, recipient)
	if err != nil {
		return err
	}
	if !o.Changed {
		_, err := fmt.Fprintf(env.Stdout, "%s is not a recipient; nothing changed\n", o.Recipient)
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "removed %s\n  %s and %s updated\n", o.Recipient, o.File, o.Config)
	if o.AnchorKept {
		fmt.Fprintf(&b, "  note: its anchor definition is still in %s; delete it by hand if nothing else uses it\n", o.Config)
	}
	b.WriteString("\nWARNING: rotate every value. The removed key could decrypt them until now,\n")
	b.WriteString("and the old file stays readable in version control history. Keys to rotate:\n")
	for _, name := range o.Names {
		fmt.Fprintf(&b, "  %s\n", name)
	}
	_, err = fmt.Fprint(env.Stdout, b.String())
	return err
}
