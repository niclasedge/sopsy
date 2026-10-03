# Design

## Context

Greenfield repository. All later changes build on the packages introduced here.
Overall design: `docs/design.md`.

## Goals / Non-Goals

**Goals:**
- One internal API for "open secrets file, decrypt, modify, encrypt, write" that every command reuses.
- Identical behavior on macOS, Linux and Windows.

**Non-Goals:**
- Calling external `sops`/`age` binaries.
- File formats other than dotenv.
- Key protection beyond file permissions (passphrase, keychain, hardware).

## Decisions

- **SOPS as a library, not a subprocess.** Use `github.com/getsops/sops/v3` packages
  (dotenv store, `aes` cipher, `keyservice` local client, `config` for `.sops.yaml`)
  — the same packages the `sops` CLI uses, so output is format-identical.
  Alternative: shell out to `sops` — rejected, would require an install on every
  machine, which is the problem sopsy solves.
- **age via `filippo.io/age`** for key generation and identity parsing.
- **Package layout:** `internal/keys` (lookup, generate), `internal/store`
  (decrypt/encrypt/atomic write), `internal/cli` (commands, exit codes). `main.go`
  only wires `internal/cli`.
- **CLI framework:** standard library `flag` with a small subcommand dispatcher.
  Alternative: cobra — rejected for v1, the command set is small.
- **Key file on Windows:** written into `%AppData%`; Go's `chmod` cannot express
  ACLs there, the profile directory's default ACL is the protection.
- **CI from day one:** GitHub Actions matrix ubuntu/macos/windows running
  `go vet` and `go test ./...`.

## Risks / Trade-offs

- [SOPS library internals change between versions] → pin the version in `go.mod`;
  compatibility test against the `sops` CLI in CI catches format drift.
- [Binary size grows with SOPS' cloud key-source dependencies] → accept for v1; it
  keeps group access via KMS/Vault possible later without code changes.
- [`os.UserConfigDir` differs from what users expect on macOS] → the `~/.config`
  lookup step covers existing setups; `init` writes to the SOPS default.

## Implementation notes

- **Custom in-process key service.** The SOPS `keyservice` local client is used
  with a custom server (`internal/store/keyservice.go`) that only handles age and
  only the identities found by `internal/keys`. The stock server would load
  identities from `SOPS_AGE_KEY`, `SOPS_AGE_KEY_CMD` and `~/.ssh` on its own — a
  silent second key lookup that contradicts the lookup order in the spec.
- **macOS default key location honours `$XDG_CONFIG_HOME`**, as SOPS does, so a
  bare `sops` finds a key created by `sopsy init`.
- **CRLF tolerated on read.** A Windows checkout with `core.autocrlf` turns the
  committed file into CRLF, which the SOPS dotenv parser cannot read. The MAC
  covers values, not line endings, so CRLF is read as LF.
- **Malformed lines are reported before the SOPS parser runs**, because its
  error message quotes the offending line.
