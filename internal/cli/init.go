package cli

import (
	"fmt"

	"github.com/niclasedge/sopsy/internal/setup"
)

func runInit(env Env, args []string) error {
	fs := flags(env, "init")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return &Error{Msg: "init takes no arguments", Hint: []string{"usage: sopsy init [--file F]"}}
	}
	secrets := env.secretsFile(*file)

	k, res, err := setup.Key(env.Keys)
	if err != nil {
		return err
	}
	printStep(env, "age key", res)

	res, err = setup.Config(env.Dir, secrets, k)
	if err != nil {
		return err
	}
	printStep(env, ".sops.yaml", res)

	res, err = setup.SecretsFile(res.Path, secrets)
	if err != nil {
		return err
	}
	printStep(env, "secrets file", res)
	return nil
}

func printStep(env Env, step string, r setup.Result) {
	status := "present"
	if r.Created {
		status = "created"
	}
	line := fmt.Sprintf("%-8s %-13s %s", status, step, r.Path)
	if r.Detail != "" {
		line += " (" + r.Detail + ")"
	}
	_, _ = fmt.Fprintln(env.Stdout, line)
}
