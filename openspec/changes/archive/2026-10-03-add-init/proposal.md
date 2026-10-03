# Proposal

## Why

Setting up SOPS + age today means installing two tools, generating a key in the
right place (SOPS on macOS does not look in `~/.config`), hand-writing
`.sops.yaml` and creating the first encrypted file. Every later command needs the
same foundation — finding the key, finding the config, failing loudly — so it is
built first. See `docs/design.md`.

## What Changes

- New Go module `github.com/niclasedge/sopsy` with SOPS and age compiled in.
- Age key lookup that works on macOS, Linux and Windows, including the macOS
  `~/.config` location SOPS itself ignores.
- CLI conventions shared by all commands: exit code 125 for sopsy's own
  failures, actionable error messages, no secret value in any output.
- `sopsy init`: idempotent setup — create only what is missing (age key,
  `.sops.yaml`, empty encrypted secrets file).
- CI on ubuntu, macos and windows from the first commit.

## Capabilities

### New Capabilities
- `key-discovery`: where sopsy looks for the age private key and the secrets file, and in which order.
- `cli-conventions`: exit codes, error reporting and the never-print-a-value rule shared by every command.
- `setup`: the `sopsy init` command.

### Modified Capabilities
<!-- none -->

## Impact

New repository code: `go.mod`, `main.go`, `internal/` packages for key lookup,
SOPS file access and CLI wiring. Dependencies: `github.com/getsops/sops/v3`,
`filippo.io/age`. GitHub Actions workflow for tests.
