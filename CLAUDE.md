# sopsy — working guidelines

Single Go binary (macOS, Linux, Windows): CLI plus local web UI for a
SOPS + age encrypted dotenv file. SOPS and age are compiled in; no external
binaries are called. Approved design: [`docs/design.md`](docs/design.md).

**This repository is public.** Everything committed — code, tests, fixtures,
commit messages, PR text, issue comments — is published.

## Commands

| Purpose | Command |
|---|---|
| Format + lint | `gofmt -l . && golangci-lint run ./...` |
| Vet | `go vet ./...` |
| Tests (as CI) | `SOPSY_REQUIRE_SOPS=1 go test ./...` |
| Validate a change | `openspec validate <name> --strict` |

All four must pass before pushing.

## Hard rules: secrets

1. **Never print a secret value** — not on stdout, stderr, in logs, error
   messages, HTTP responses or test failure output. Errors name the file and
   line, never the content.
2. **Values never travel in argv.** Input comes from a hidden prompt, stdin or
   a POST body.
3. **No hidden key lookup.** Only the key file found by `internal/keys` is
   used; never the SOPS default key service.

## Hard rules: behaviour

4. **Loud errors, no silent fallback.** Every failure says what is wrong and
   the next command that fixes it; never continue after a failed decryption.
5. **Exit codes follow `env(1)`:** 125 sopsy failed (incl. invalid arguments),
   126 not executable, 127 not found, else the child's code; `doctor`: 1 = a
   check failed.
6. **Files stay 100 % SOPS-compatible.** Every write goes through the SOPS
   library; the `sops` CLI must keep working on every file sopsy touches.

## Hard rules: repo

- No real secrets, keys, tokens, personal paths, hostnames or internal URLs in
  code, tests, fixtures, logs or PR text. No private key except the test-only one.
- Do not weaken `.gitignore`; Actions pinned to a SHA, least-privilege
  `permissions`. Security-relevant behaviour gets a test that fails without it.
- Work = one issue + one OpenSpec change, branch `change/<name>`, PR,
  squash-merge after green CI. English, Conventional Commits.
- Pushing a tag publishes a release — only with explicit approval.

## Entry

World pack: `sopsy-world/world.md#coverage-map`. Read only the section the task needs.
