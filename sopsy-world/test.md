# Testing

## Auf einen Blick

`just test` runs the suite; `just lint` checks without autofix; `just cov` writes `coverage/lcov.info`. File arguments select test or lint targets.

Tests prove that no secret leaks, that nothing touches the developer's real
home or key, and that files stay readable by the real `sops` CLI in both
directions. CLI tests run through one harness; web UI tests call the handler
directly.

## No secret in output

Every CLI test runs through the `world` harness in
`internal/cli/harness_test.go`, which calls `testutil.AssertNoSecret` on
stdout and stderr. Any value that stands for a secret in a test must be listed
in `testutil.Secrets`.

Wo: `internal/cli/harness_test.go`, `internal/testutil/testutil.go`

## Isolation

Tests never touch the developer's real home, key or `.sops.yaml`. The harness
fakes home, working directory and environment; key lookup takes an injectable
`keys.Env`, so all three OS layouts are tested on any host.

Wo: `internal/cli/harness_test.go`, `internal/keys/keys.go`, `internal/keys/keys_test.go`

## Sops compatibility

Compatibility both ways: `internal/testutil/testdata/sops-cli.env` was
encrypted by the real `sops` CLI; files written by sopsy are decrypted with
`sops` in tests (`testutil.SopsDecrypt`). Locally these tests skip without
`sops`; CI sets `SOPSY_REQUIRE_SOPS=1` so they cannot skip there.

Wo: `internal/testutil/testdata/sops-cli.env`, `internal/testutil/testutil.go`, `.github/workflows/test.yml`

## Test-only key

`internal/testutil/testdata/test-only.agekey` is public on purpose and
protects nothing but fixtures. Never use it for anything else and never commit
any other private key.

Wo: `internal/testutil/testdata/test-only.agekey`

## Web UI tests

`internal/web` tests send requests straight to the handler (`httptest`) and run
every response body through `AssertNoSecret`; each protection (Host check,
session, CSRF, headers, log) has a test that fails when the protection is
removed.

Belege: #6

Wo: `internal/web/server_test.go`, `internal/web/pages_test.go`, `internal/web/helpers_test.go`

## Test style

Prefer table tests; test behaviour through the CLI where the spec describes
CLI behaviour, and through the package where it describes package behaviour.

Wo: `internal/cli/`, `openspec/specs/`
