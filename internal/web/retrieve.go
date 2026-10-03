package web

import (
	"path/filepath"
	"strings"
)

// exampleInput is what the Retrieve page's commands are built from.
type exampleInput struct {
	Dir, File, Executable, KeyFile string
	Names                          []string
}

// example is one shell's way to use the secrets.
type example struct {
	Shell string
	Lines []string
	Note  string
}

func examples(in exampleInput) []example {
	name := "API_TOKEN"
	if len(in.Names) > 0 {
		name = in.Names[0]
	}
	exe := in.Executable
	if exe == "" {
		exe = "sopsy"
	}
	sh := "sopsy run --file " + posixQuote(in.File) + " -- "
	cron := "*/15 * * * * cd " + posixQuote(in.Dir) + " && "
	if in.KeyFile != "" {
		cron += "SOPS_AGE_KEY_FILE=" + posixQuote(in.KeyFile) + " "
	}
	cron += posixQuote(exe) + " run --file " + posixQuote(in.File) + " -- ./job.sh"
	return []example{
		{
			Shell: "bash / zsh",
			Lines: []string{
				sh + "./deploy.sh",
				sh + `sh -c 'curl -fsS -H "Authorization: Bearer $` + name + `" https://api.example.com/'`,
			},
			Note: "The command gets every key as an environment variable, e.g. $" + name + ".",
		},
		{
			Shell: "PowerShell",
			Lines: []string{"sopsy run --file " + psQuote(in.File) + ` -- pwsh -NoProfile -File .\deploy.ps1`},
			Note:  "Inside the script: $env:" + name,
		},
		{
			Shell: "cmd",
			Lines: []string{"sopsy run --file " + `"` + in.File + `"` + " -- cmd /c deploy.bat"},
			Note:  "Inside the batch file: %" + name + "%",
		},
		{
			Shell: "cron",
			// % ends the command in a crontab line unless escaped.
			Lines: []string{strings.ReplaceAll(cron, "%", `\%`)},
			Note:  "cron does not read your shell profile, so the line uses absolute paths and names the key file.",
		},
	}
}

func posixQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// initCommand is the CLI equivalent of the setup steps.
func initCommand(cfg Config) string {
	if cfg.SecretsFile == filepath.Join(cfg.Dir, "secrets.env") {
		return "sopsy init"
	}
	return "sopsy init --file " + posixQuote(cfg.SecretsFile)
}
