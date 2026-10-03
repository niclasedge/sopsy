#!/bin/sh
# Writes THIRD_PARTY_LICENSES for the release archives: every compiled-in
# dependency with its license text, plus the Go standard library. Fails when
# go-licenses cannot identify a license or finds a forbidden one.
set -eu

licenses="go run github.com/google/go-licenses/v2@v2.0.1"
self=github.com/niclasedge/sopsy
out=THIRD_PARTY_LICENSES

# Official Go archives ship LICENSE in GOROOT; Homebrew moves it one level up.
goroot=$(go env GOROOT)
golicense=$goroot/LICENSE
[ -f "$golicense" ] || golicense=$goroot/../LICENSE
[ -f "$golicense" ] || { echo "Go LICENSE not found next to $goroot" >&2; exit 1; }

$licenses check . --ignore "$self"
$licenses report . --ignore "$self" --template build/third-party-licenses.tmpl > "$out.tmp"
{
	printf '\n%s\n' "================================================================================"
	printf '%s\n' "Go standard library and runtime $(go env GOVERSION)"
	printf '%s\n\n' "License: BSD-3-Clause (https://go.dev/LICENSE)"
	cat "$golicense"
} >> "$out.tmp"
mv "$out.tmp" "$out"
