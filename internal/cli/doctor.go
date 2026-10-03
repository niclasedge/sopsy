package cli

import (
	"fmt"
	"strings"

	"github.com/niclasedge/sopsy/internal/doctor"
)

// ExitCheckFailed is doctor's exit code when a check failed. It differs from
// ExitFail on purpose: doctor itself worked, it found a problem.
const ExitCheckFailed = 1

func runDoctor(env Env, args []string) error {
	fs := flags(env, "doctor")
	file := fileFlag(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return &Error{Msg: "doctor takes no arguments", Hint: []string{"usage: sopsy doctor [--file F]"}}
	}
	results := doctor.Run(doctor.Input{Keys: env.Keys, Dir: env.Dir, SecretsFile: env.secretsFile(*file)})
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "%-8s %-16s %s\n", r.Status, r.Name, r.Detail)
		if r.Fix != "" {
			fmt.Fprintf(&b, "%-8s %-16s fix: %s\n", "", "", r.Fix)
		}
	}
	if _, err := fmt.Fprint(env.Stdout, b.String()); err != nil {
		return err
	}
	if doctor.Failed(results) {
		return &exitCode{ExitCheckFailed}
	}
	return nil
}
