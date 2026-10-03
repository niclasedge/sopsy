package cli

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
)

// Release builds set these with -ldflags "-X"; see .goreleaser.yaml.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// releaseVersion matches a tagged module version such as v0.1.0 or
// v0.1.0-rc.1, but not pseudo-versions or +dirty builds.
var releaseVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`)

func runVersion(env Env, args []string) error {
	fs := flags(env, "version")
	if err := parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return &Error{Msg: "version takes no arguments", Hint: []string{"usage: sopsy version"}}
	}
	v := version
	if v == "dev" {
		// `go install github.com/niclasedge/sopsy@v0.1.0` has no ldflags but
		// records the module version.
		if info, ok := debug.ReadBuildInfo(); ok && releaseVersion.MatchString(info.Main.Version) {
			v = info.Main.Version
		}
	}
	_, err := fmt.Fprintln(env.Stdout, versionLine(v, commit, date))
	return err
}

func versionLine(version, commit, date string) string {
	s := "sopsy " + strings.TrimPrefix(version, "v")
	var meta []string
	if commit != "" {
		meta = append(meta, "commit "+commit)
	}
	if date != "" {
		meta = append(meta, "built "+date)
	}
	if len(meta) > 0 {
		s += " (" + strings.Join(meta, ", ") + ")"
	}
	return s
}
