package cli

import (
	"errors"
	"runtime"

	"github.com/niclasedge/sopsy/internal/runner"
)

func runRun(env Env, args []string) error {
	fs := flags(env, "run")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return &Error{Msg: "missing command", Hint: []string{"usage: sopsy run [--file F] -- COMMAND [ARGS...]"}}
	}
	// Decrypt before resolving the command, so a missing command cannot
	// hide a broken key setup.
	plain, err := env.decrypt(env.secretsFile(*file))
	if err != nil {
		return err
	}
	var vars []string
	for _, e := range plain.Entries() {
		vars = append(vars, e.Name+"="+e.Value)
	}
	code, err := runner.Run(runner.Command{
		Args:   fs.Args(),
		Env:    runner.MergeEnv(env.Environ(), vars, runtime.GOOS == "windows"),
		Stdin:  env.Stdin,
		Stdout: env.Stdout,
		Stderr: env.Stderr,
	})
	var se *runner.StartError
	if errors.As(err, &se) {
		return &Error{Msg: se.Error(), Code: se.Code}
	}
	if err != nil {
		return err
	}
	return &exitCode{code}
}
