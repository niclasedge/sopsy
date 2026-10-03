// Package setup implements the steps of `sopsy init`. Each step creates only
// what is missing; the CLI and the web UI run the same steps.
package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/getsops/sops/v3/config"

	"github.com/niclasedge/sopsy/internal/keys"
	"github.com/niclasedge/sopsy/internal/sopsconfig"
	"github.com/niclasedge/sopsy/internal/store"
)

// Result is the outcome of one step.
type Result struct {
	Created bool
	Path    string
	Detail  string
}

// Key loads the age key, or generates one at the SOPS default location when
// none exists. An explicitly configured but missing key file is an error.
func Key(env keys.Env) (*keys.Key, Result, error) {
	k, err := env.Load()
	if err == nil {
		return k, Result{Path: k.Path, Detail: "public key " + k.Public}, nil
	}
	var nf *keys.NotFoundError
	if !errors.As(err, &nf) || nf.Explicit {
		return nil, Result{}, err
	}
	path, err := env.DefaultPath()
	if err != nil {
		return nil, Result{}, fmt.Errorf("cannot determine where to create the key: %w", err)
	}
	k, err = keys.Generate(path)
	if err != nil {
		return nil, Result{}, err
	}
	return k, Result{Created: true, Path: path, Detail: "public key " + k.Public}, nil
}

// Config finds .sops.yaml from dir upwards, or creates it in dir with the
// own key as the only recipient. For an existing file it reports whether the
// own key is a recipient of the rule for secretsFile.
func Config(dir, secretsFile string, k *keys.Key) (Result, error) {
	path, err := sopsconfig.Find(dir)
	if errors.Is(err, sopsconfig.ErrNotFound) {
		path = filepath.Join(dir, sopsconfig.FileName)
		if err := sopsconfig.Create(path, secretsFile, k.Public); err != nil {
			return Result{}, err
		}
		return Result{Created: true, Path: path, Detail: "own key is the recipient"}, nil
	}
	if err != nil {
		return Result{}, err
	}
	c, err := sopsconfig.Load(path)
	if err != nil {
		return Result{}, err
	}
	rule, err := c.RuleFor(secretsFile)
	if err != nil {
		return Result{}, err
	}
	detail := "own key is NOT a recipient"
	if slices.ContainsFunc(rule.Recipients(), k.IsRecipient) {
		detail = "own key is a recipient"
	}
	return Result{Path: path, Detail: detail}, nil
}

// SecretsFile creates an empty encrypted secrets file for the creation rule
// in configPath when it does not exist yet.
func SecretsFile(configPath, secretsFile string) (Result, error) {
	_, err := os.Stat(secretsFile)
	if err == nil {
		return Result{Path: secretsFile}, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return Result{}, err
	}
	rule, err := config.LoadCreationRuleForFile(configPath, secretsFile, nil)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", configPath, err)
	}
	if rule == nil {
		return Result{}, fmt.Errorf("%s has no creation_rules", configPath)
	}
	if err := store.Create(secretsFile, rule); err != nil {
		return Result{}, err
	}
	return Result{Created: true, Path: secretsFile, Detail: "empty"}, nil
}
