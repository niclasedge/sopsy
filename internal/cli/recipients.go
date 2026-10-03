package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/niclasedge/sopsy/internal/sopsconfig"
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

// recipientChange holds everything a recipient change touches, loaded and
// validated before anything is written.
type recipientChange struct {
	recipient string
	file      *store.File
	config    *sopsconfig.Config
	rule      *sopsconfig.Rule
	inFile    bool
	inConfig  bool
	// configChanged is set once the rule was edited in memory.
	configChanged bool
}

func loadRecipientChange(env Env, name string, args []string) (*recipientChange, error) {
	fs := flags(env, "recipients "+name)
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return nil, err
	}
	if fs.NArg() != 1 {
		return nil, &Error{Msg: name + " takes exactly one age public key", Hint: []string{
			"usage: sopsy recipients " + name + " [--file F] AGE_PUBLIC_KEY",
		}}
	}
	recipient := fs.Arg(0)
	if err := store.ValidateRecipient(recipient); err != nil {
		return nil, &Error{Msg: err.Error() + "; nothing changed", Hint: []string{
			"an age public key starts with age1; get it with `sopsy pubkey` on the machine that should decrypt",
		}}
	}
	secrets := env.secretsFile(*file)
	f, err := store.Load(secrets)
	if err != nil {
		return nil, err
	}
	confPath, err := sopsconfig.Find(env.Dir)
	if err != nil {
		return nil, err
	}
	c, err := sopsconfig.Load(confPath)
	if err != nil {
		return nil, err
	}
	rule, err := c.RuleFor(secrets)
	if err != nil {
		return nil, err
	}
	return &recipientChange{
		recipient: recipient,
		file:      f,
		config:    c,
		rule:      rule,
		inFile:    slices.Contains(f.Recipients(), recipient),
		inConfig:  slices.Contains(rule.Recipients(), recipient),
	}, nil
}

// apply writes the secrets file first and .sops.yaml second, so a failure
// in between leaves a state the error message can describe exactly.
func (ch *recipientChange) apply(env Env, add, remove []string, manual string) error {
	if len(add)+len(remove) > 0 {
		k, err := env.Keys.Load()
		if err != nil {
			return err
		}
		if err := ch.file.ChangeRecipients(k.Identities, k.Public, add, remove); err != nil {
			return err
		}
	}
	if !ch.configChanged {
		return nil
	}
	if err := ch.config.Save(); err != nil {
		return &Error{
			Msg:  fmt.Sprintf("%s was updated, but %s could not be written: %v", ch.file.Path, ch.config.Path, err),
			Hint: []string{manual},
		}
	}
	return nil
}

func recipientsAdd(env Env, args []string) error {
	ch, err := loadRecipientChange(env, "add", args)
	if err != nil {
		return err
	}
	if ch.inFile && ch.inConfig {
		_, err := fmt.Fprintf(env.Stdout, "%s is already a recipient; nothing changed\n", ch.recipient)
		return err
	}
	if !ch.inConfig {
		if err := ch.rule.Add(ch.recipient); err != nil {
			return err
		}
		ch.configChanged = true
	}
	var add []string
	if !ch.inFile {
		add = []string{ch.recipient}
	}
	manual := fmt.Sprintf("add %s to the age list in %s by hand", ch.recipient, ch.config.Path)
	if err := ch.apply(env, add, nil, manual); err != nil {
		return err
	}
	_, err = fmt.Fprintf(env.Stdout, "added %s\n  %s and %s updated\n", ch.recipient, ch.file.Path, ch.config.Path)
	return err
}

func recipientsRemove(env Env, args []string) error {
	ch, err := loadRecipientChange(env, "remove", args)
	if err != nil {
		return err
	}
	if !ch.inFile && !ch.inConfig {
		_, err := fmt.Fprintf(env.Stdout, "%s is not a recipient; nothing changed\n", ch.recipient)
		return err
	}
	if ch.inConfig && len(ch.rule.Recipients()) == 1 {
		return store.ErrLastRecipient
	}
	aliasKept := false
	if ch.inConfig {
		if _, aliasKept, err = ch.rule.Remove(ch.recipient); err != nil {
			return err
		}
		ch.configChanged = true
	}
	var remove []string
	if ch.inFile {
		remove = []string{ch.recipient}
	}
	manual := fmt.Sprintf("remove %s from the age list in %s by hand", ch.recipient, ch.config.Path)
	if err := ch.apply(env, nil, remove, manual); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "removed %s\n  %s and %s updated\n", ch.recipient, ch.file.Path, ch.config.Path)
	if aliasKept {
		fmt.Fprintf(&b, "  note: its anchor definition is still in %s; delete it by hand if nothing else uses it\n", ch.config.Path)
	}
	b.WriteString("\nWARNING: rotate every value. The removed key could decrypt them until now,\n")
	b.WriteString("and the old file stays readable in version control history. Keys to rotate:\n")
	for _, name := range ch.file.Names() {
		fmt.Fprintf(&b, "  %s\n", name)
	}
	_, err = fmt.Fprint(env.Stdout, b.String())
	return err
}
