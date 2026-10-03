# sopsy — working guidelines

Single Go binary (macOS, Linux, Windows): CLI plus local web UI for a
SOPS + age encrypted dotenv file. SOPS and age are compiled in; no external
binaries are called. Approved design: [`docs/design.md`](docs/design.md).

**This repository is public.** Everything committed — code, tests, fixtures,
commit messages, PR text, issue comments — is published.

## Hard rules

These come from the design and apply to every change, CLI and web UI alike.

1. **Never print a secret value** — not on stdout, stderr, in logs, error
   messages, HTTP responses or test failure output. Errors name the file and
   line, never the content.
2. **Values never travel in argv.** Input comes from a hidden prompt, stdin or
   a POST body.
3. **Loud errors, no silent fallback.** Every failure says what is wrong and the
   next command that fixes it. Never continue with defaults or an empty
   environment after a failed decryption.
4. **Exit codes follow `env(1)`:** 125 sopsy itself failed (including invalid
   arguments), 126 command not executable, 127 command not found, otherwise the
   child's code. `doctor` is the documented exception: 1 = a check failed.
5. **Files stay 100 % SOPS-compatible.** Every write goes through the SOPS
   library; the `sops` CLI must keep working on every file sopsy touches.
6. **No hidden key lookup.** Only the key file found by `internal/keys` is
   used. The SOPS default key service (which reads env vars and `~/.ssh`) is
   never used — `internal/store` has its own in-process age key service.

## Workflow: issue → change → PR

Every piece of work is one GitHub issue **and** one OpenSpec change under
`openspec/changes/<name>/` (proposal, design, specs, tasks). The issue links the
change; the change is the source of truth for scope.

1. **Propose.** New work starts as an issue plus `openspec propose` (or
   `/opsx:propose`). Scope and design are agreed before code is written.
   Every task names its verification (test, command or observable behaviour).
2. **Branch.** `change/<name>` from an up-to-date `main`. One change per
   branch, one branch per PR. Respect the dependency order in the issues.
3. **Implement** the tasks in order. Tick a task (`- [x]`) only after its
   named verification passed. If reality differs from the design, update the
   change's `design.md`/specs in the same branch and say so in the PR — never
   silently drift.
4. **Local gates** (all must pass before pushing):
   ```bash
   gofmt -l . && golangci-lint run ./...
   go vet ./...
   SOPSY_REQUIRE_SOPS=1 go test ./...
   openspec validate <name> --strict
   ```
5. **Archive** the change as the last commit of the branch:
   `openspec archive <name> --yes` — this moves the delta specs into
   `openspec/specs/`, so `main` always has specs that match its code.
6. **Pull request.** Title in Conventional Commits form, e.g.
   `feat(run): start a command with secrets in its environment`. Body:
   `Closes #N`, link to the change, verification evidence per task, and any
   deviation from the design or a task that could only be verified partially
   (with a follow-up issue).
7. **CI gate.** The `test` workflow must be green on ubuntu, macos and windows,
   plus `lint`. Then squash-merge and delete the branch.

Tasks that need a manual check on a platform we cannot reach are not ticked
silently: record what was verified where, and open a follow-up issue for the
rest.

## Software stack

| Area | Choice |
|---|---|
| Language | Go (version in `go.mod`), `CGO_ENABLED=0` for releases |
| SOPS | `github.com/getsops/sops/v3` as a library (dotenv store, aes, config, keyservice types) |
| age | `filippo.io/age` |
| Hidden prompt | `golang.org/x/term` |
| `.sops.yaml` edits | `go.yaml.in/yaml/v3` on `yaml.Node` (keeps comments and anchors) |
| CLI | standard library `flag` with a small dispatcher — no cobra |
| Web UI | `net/http` + `html/template` + `go:embed`; no JS framework, no build step |
| Lint | `golangci-lint` v2 (`.golangci.yml`) |
| CI | GitHub Actions: `test.yml` (3 OS matrix + lint), `release.yml` (goreleaser on `v*` tags) |
| Release | goreleaser, six targets, checksums, `THIRD_PARTY_LICENSES` via `go-licenses` |

Add a dependency only when the standard library cannot reasonably do the job,
and check its license (it ends up in `THIRD_PARTY_LICENSES`).

## Code layout

```
main.go                  wires internal/cli only
internal/cli             commands, flags, exit codes, error → hint mapping
internal/keys            age key lookup per OS, load, generate
internal/sopsconfig      find/read/edit .sops.yaml (yaml.Node)
internal/store           load, decrypt, encrypt, atomic write of the secrets file
internal/setup           the idempotent steps of `sopsy init` (CLI and UI)
internal/testutil        fixtures, test-only key, no-secret assertion, sops CLI helpers
```

Later changes add `internal/runner` (process start, signals), `internal/doctor`
(check list) and `internal/web` (UI). Platform-specific code lives in
`_unix.go` / `_windows.go` files with build tags, never in runtime `if`s where
a build tag fits.

## Testing

- **No secret in output:** every CLI test runs through the `world` harness in
  `internal/cli/harness_test.go`, which calls `testutil.AssertNoSecret` on
  stdout and stderr. Any value that stands for a secret in a test must be listed
  in `testutil.Secrets`.
- **Isolation:** tests never touch the developer's real home, key or
  `.sops.yaml`. The harness fakes home, working directory and environment;
  key lookup takes an injectable `keys.Env`, so all three OS layouts are tested
  on any host.
- **Compatibility both ways:** `internal/testutil/testdata/sops-cli.env` was
  encrypted by the real `sops` CLI; files written by sopsy are decrypted with
  `sops` in tests (`testutil.SopsDecrypt`). Locally these tests skip without
  `sops`; CI sets `SOPSY_REQUIRE_SOPS=1` so they cannot skip there.
- **Test-only key:** `internal/testutil/testdata/test-only.agekey` is public on
  purpose and protects nothing but fixtures. Never use it for anything else and
  never commit any other private key.
- Prefer table tests; test behaviour through the CLI where the spec describes
  CLI behaviour, and through the package where it describes package behaviour.

## Public-repo hygiene

- No real secrets, keys, tokens, personal paths, hostnames or internal URLs in
  code, tests, fixtures, logs or PR text.
- `.gitignore` blocks `keys.txt`, `.env`, `*.dec`, `*.plain`; do not weaken it.
- GitHub Actions are pinned to a commit SHA (with the version as a comment);
  downloaded tools are checksum-verified. Workflows use least-privilege
  `permissions`.
- Security-relevant behaviour (web UI hardening, file permissions, key lookup)
  always gets a test that would fail if the protection were removed.

## Conventions

- Code, comments, commit messages, issues and PRs in English.
- Conventional Commits (`feat`, `fix`, `docs`, `test`, `ci`, `chore`, `refactor`),
  scope = capability or package.
- Comments explain *why*; the code says *what*. Match the density of the
  surrounding code.
- Keep changes surgical: a PR does what its change says and nothing else.
