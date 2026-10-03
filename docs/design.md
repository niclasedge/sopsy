# sopsy — design (v1)

Status: approved 2026-10-03. Implementation is tracked as OpenSpec changes under
`openspec/changes/` and as GitHub issues.

## What sopsy is

A single binary for macOS, Linux and Windows that manages a
[SOPS](https://github.com/getsops/sops)-encrypted dotenv file with
[age](https://age-encryption.org/) keys — a CLI plus a local, guided web UI.

It replaces having to install, configure and remember `sops` and `age`. SOPS and
age are compiled in as Go libraries; nothing else has to be installed. The files
it reads and writes stay 100 % SOPS-compatible, so the regular `sops` CLI keeps
working side by side.

Inspired by [simonw/llm-keys-ui](https://github.com/simonw/llm-keys-ui): values
can be **set** without passing through a terminal, a shell history or an AI
agent's context — and sopsy never prints a value back.

## Commands

| Command | Does |
|---|---|
| `sopsy init` | Idempotent setup: create the age key, `.sops.yaml` with the own recipient, and an empty encrypted file — only what is missing |
| `sopsy run -- <cmd>` | Decrypt into the environment of `<cmd>` (all keys), pass through exit code and signals — like `sops exec-env` |
| `sopsy set KEY` / `sopsy unset KEY` | Value from a hidden prompt or stdin, never from argv |
| `sopsy keys` | List key names without decrypting |
| `sopsy pubkey` | Print the own age public key |
| `sopsy recipients add <age1…>` / `remove` | Change who can decrypt, re-encrypt the data key |
| `sopsy doctor` | Key found? Decrypt works? Key file permissions? `sops`/`age` installed (info only)? |
| `sopsy ui` | Local web UI |

Deliberately absent in v1: `get`/`view` (no value is ever printed — `sops -d`
still works because the format is compatible), `edit` (would write plaintext to a
temp file), per-app key filtering, group access.

## Files

- **Secrets file**: `--file`, else `$SOPSY_FILE`, else `secrets.env`. Dotenv only in v1.
- **`.sops.yaml`**: searched from the working directory upwards, like SOPS.
- **age key** lookup order:
  1. `$SOPS_AGE_KEY_FILE`
  2. `~/.config/sops/age/keys.txt`
  3. SOPS' own default, `os.UserConfigDir()/sops/age/keys.txt`
     (macOS `~/Library/Application Support/…`, Windows `%AppData%\sops\age\keys.txt`)

  Step 2 exists because SOPS on macOS does *not* look in `~/.config`, and the
  resulting error ("no identity matched any of the recipients") blames the
  recipient list instead of the lookup path. `init` writes to location 3 so a
  bare `sops` finds the key without any environment variable.

## Security model

The only lock is the **private age key file**. Anyone can run `sopsy`; only a
holder of a private key listed as a recipient can decrypt.

| Who | Can read values? | Why |
|---|---|---|
| another OS user | no, while the key file is `0600` / profile-protected | cannot read the key |
| any process running as the same user | yes | can read the key file |
| root / Administrator | yes | file permissions do not bind them |
| someone with repo access but no key | no | sees key names, not values |

Revoking a recipient means: remove it, re-encrypt **and rotate the values** —
the old ciphertext is still in git history.

Environment variables (what `run` produces) protect against accidental output,
not against an attacker already running as the same user
(`/proc/<pid>/environ`, `ps eww`).

## Web UI

`sopsy ui` binds `127.0.0.1` on a free port and opens the browser with a one-time
token in the URL.

Pages:
1. **Setup** — while anything is missing: key → `.sops.yaml` → secrets file →
   test decrypt, each step showing the equivalent CLI command.
2. **Keys** — names and "set" state; set/overwrite, delete (confirm), add.
   Values are never displayed.
3. **Recipients** — who can decrypt; own public key with copy button; add;
   remove with a "rotate your secrets" warning.
4. **Retrieve** — example commands for bash/zsh, PowerShell, cmd, cron.

Protection: Host header allowlist (DNS rebinding — llm-keys-ui lacks this),
token exchanged for an `HttpOnly; SameSite=Strict` cookie, CSRF token per form,
strict CSP, `Cache-Control: no-store`, idle shutdown after 15 minutes, request
bodies never logged. Plain `net/http` + `html/template` + `go:embed`; no JS
framework, no build step.

## Data flow and errors

- **Write path**: decrypt in memory → change → re-encrypt through the SOPS
  library (data key kept, MAC recomputed) → atomic write (temp file in the same
  directory + rename). The file hash is compared before writing; if the file
  changed in between (UI and CLI at once), sopsy aborts instead of overwriting.
- **`run`**: values from the file override existing environment variables (same
  as `sops exec-env`). Signals and the child's exit code are passed through. On
  Windows there is no `exec`; the child is spawned and Ctrl+C forwarded.
- **Exit codes** (convention of `env(1)`): 125 sopsy itself failed, 126 command
  not executable, 127 command not found, otherwise the child's code.
- **Errors are loud and actionable**; there is never a silent fallback. A value
  never appears in any message, including parse errors.

## Tests and release

- Unit: key lookup per OS (table tests with faked `HOME`/`AppData`), set/unset,
  recipients roundtrip.
- Compatibility both ways: files written by sopsy decrypt with the real `sops`
  binary and vice versa, using a committed test-only age key.
- UI via `httptest`: wrong Host → 403, no token → 403, no CSRF → 403, CSP
  present, no test value in any response body.
- CI: GitHub Actions on ubuntu, macos, windows — `go test`, `go vet`, end-to-end
  `init → set → run -- printenv`.
- Release: goreleaser on tag for darwin/linux/windows × amd64/arm64, checksums,
  `THIRD_PARTY_LICENSES` (SOPS is MPL-2.0: binary distribution must point to its
  source). Install via `go install github.com/niclasedge/sopsy@latest` or a
  release download.
- Not in v1: Apple notarization, Windows code signing, Homebrew, winget.

## Done when

1. CI is green on all three operating systems.
2. An existing SOPS dotenv file encrypted by the `sops` CLI can be listed with
   `sopsy keys` and used with `sopsy run` unchanged.

## Later

- Group access on Windows via DPAPI-NG (AD group SID as recipient) — see issue.
- Per-app key filtering (`--as <app>` with an allowlist).
- Key protection options: passphrase, OS keychain, hardware keys via age plugins.
