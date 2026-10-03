// Package recipients adds and removes age recipients in the secrets file and
// in its .sops.yaml rule together. The CLI and the web UI share it, so both
// apply the same validation and guards.
package recipients

import (
	"fmt"
	"slices"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/store"
)

// Outcome describes what Add or Remove did.
type Outcome struct {
	Recipient string
	// Changed is false when there was nothing to do.
	Changed bool
	File    string
	Config  string
	// AnchorKept is set when a removed entry was an alias: its anchor
	// definition stays in .sops.yaml.
	AnchorKept bool
	// Names lists the keys in the file; after a removal every one of them
	// should be rotated.
	Names []string
}

// InvalidError reports a malformed recipient; nothing was changed.
type InvalidError struct{ Err error }

func (e *InvalidError) Error() string { return e.Err.Error() + "; nothing changed" }
func (e *InvalidError) Unwrap() error { return e.Err }

// PartialError reports that the secrets file was updated but .sops.yaml
// could not be written. Manual says how to finish by hand.
type PartialError struct {
	File, Config, Manual string
	Err                  error
}

func (e *PartialError) Error() string {
	return fmt.Sprintf("%s was updated, but %s could not be written: %v", e.File, e.Config, e.Err)
}
func (e *PartialError) Unwrap() error { return e.Err }

// change holds everything a recipient change touches, loaded and validated
// before anything is written.
type change struct {
	recipient string
	file      *store.File
	config    *sopsconfig.Config
	rule      *sopsconfig.Rule
	inFile    bool
	inConfig  bool
	// configChanged is set once the rule was edited in memory.
	configChanged bool
}

func load(dir, secretsFile, recipient string) (*change, error) {
	if err := store.ValidateRecipient(recipient); err != nil {
		return nil, &InvalidError{err}
	}
	f, err := store.Load(secretsFile)
	if err != nil {
		return nil, err
	}
	confPath, err := sopsconfig.Find(dir)
	if err != nil {
		return nil, err
	}
	c, err := sopsconfig.Load(confPath)
	if err != nil {
		return nil, err
	}
	rule, err := c.RuleFor(secretsFile)
	if err != nil {
		return nil, err
	}
	return &change{
		recipient: recipient,
		file:      f,
		config:    c,
		rule:      rule,
		inFile:    slices.Contains(f.Recipients(), recipient),
		inConfig:  slices.Contains(rule.Recipients(), recipient),
	}, nil
}

func (ch *change) outcome(changed bool) *Outcome {
	return &Outcome{
		Recipient: ch.recipient,
		Changed:   changed,
		File:      ch.file.Path,
		Config:    ch.config.Path,
		Names:     ch.file.Names(),
	}
}

// apply writes the secrets file first and .sops.yaml second, so a failure
// in between leaves a state the error can describe exactly.
func (ch *change) apply(ke keys.Env, add, remove []string, manual string) error {
	if len(add)+len(remove) > 0 {
		k, err := ke.Load()
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
		return &PartialError{File: ch.file.Path, Config: ch.config.Path, Manual: manual, Err: err}
	}
	return nil
}

// Add makes recipient able to decrypt the secrets file and lists it in the
// .sops.yaml rule. Whichever of the two already has it is left alone.
func Add(ke keys.Env, dir, secretsFile, recipient string) (*Outcome, error) {
	ch, err := load(dir, secretsFile, recipient)
	if err != nil {
		return nil, err
	}
	if ch.inFile && ch.inConfig {
		return ch.outcome(false), nil
	}
	if !ch.inConfig {
		if err := ch.rule.Add(ch.recipient); err != nil {
			return nil, err
		}
		ch.configChanged = true
	}
	var add []string
	if !ch.inFile {
		add = []string{ch.recipient}
	}
	manual := fmt.Sprintf("add %s to the age list in %s by hand", ch.recipient, ch.config.Path)
	if err := ch.apply(ke, add, nil, manual); err != nil {
		return nil, err
	}
	return ch.outcome(true), nil
}

// Remove takes recipient out of the secrets file and the .sops.yaml rule.
// It refuses to remove the last recipient.
func Remove(ke keys.Env, dir, secretsFile, recipient string) (*Outcome, error) {
	ch, err := load(dir, secretsFile, recipient)
	if err != nil {
		return nil, err
	}
	if !ch.inFile && !ch.inConfig {
		return ch.outcome(false), nil
	}
	if ch.inConfig && len(ch.rule.Recipients()) == 1 {
		return nil, store.ErrLastRecipient
	}
	anchorKept := false
	if ch.inConfig {
		if _, anchorKept, err = ch.rule.Remove(ch.recipient); err != nil {
			return nil, err
		}
		ch.configChanged = true
	}
	var remove []string
	if ch.inFile {
		remove = []string{ch.recipient}
	}
	manual := fmt.Sprintf("remove %s from the age list in %s by hand", ch.recipient, ch.config.Path)
	if err := ch.apply(ke, nil, remove, manual); err != nil {
		return nil, err
	}
	o := ch.outcome(true)
	o.AnchorKept = anchorKept
	return o, nil
}
