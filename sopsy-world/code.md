# Code layout

## Auf einen Blick

`main.go` only wires `internal/cli`; everything else is a package under
`internal/`. Shared logic (setup, doctor, recipients, hints) serves both the
CLI and the web UI. Platform differences are build-tagged files.

## Packages

```
main.go                  wires internal/cli only
internal/cli             commands, flags, exit codes, error → hint mapping
internal/keys            age key lookup per OS, load, generate
internal/sopsconfig      find/read/edit .sops.yaml (yaml.Node)
internal/store           load, decrypt, encrypt, recipients of the secrets file
internal/atomicfile      temp file + rename, shared by store and sopsconfig
internal/setup           the idempotent steps of `sopsy init` (CLI and UI)
internal/runner          process start, environment merge, signal forwarding
internal/doctor          the check list behind `sopsy doctor` (CLI and UI)
internal/recipients      add/remove recipients in file and .sops.yaml (CLI and UI)
internal/hint            error → message + fix steps (CLI and UI)
internal/web             `sopsy ui`: server, security middleware, embedded pages
internal/testutil        fixtures, test-only key, no-secret assertion, sops CLI helpers
build/                   THIRD_PARTY_LICENSES generator used by goreleaser
```

Live inventory: `go list ./...`.

Wo: `main.go`, `internal/`, `build/`

## Platform code

Platform-specific code lives in `_unix.go` / `_windows.go` (or `_darwin.go`)
files with build tags, never in runtime `if`s where a build tag fits.

Wo: `internal/web/browser_darwin.go`, `internal/web/browser_windows.go`, `internal/web/browser_other.go`

## Key lookup

Only the key file found by `internal/keys` is used. The SOPS default key
service (which reads env vars and `~/.ssh`) is never used — `internal/store`
has its own in-process age key service.

Wo: `internal/keys/keys.go`, `internal/store/`

## Errors and exit codes

The hard rules come from the design and apply to every change, CLI and web UI
alike. Every failure says what is wrong and the next command that fixes it.
Never continue with defaults or an empty environment after a failed
decryption. Exit codes follow `env(1)`; `doctor` is the documented exception.

Wo: `internal/cli/`, `internal/hint/`, `docs/design.md`
