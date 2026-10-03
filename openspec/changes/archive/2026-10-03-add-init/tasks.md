# Tasks

## 1. Module and CI

- [x] 1.1 Create `go.mod` (`github.com/niclasedge/sopsy`), `main.go` and `internal/cli` dispatcher printing usage; verify `go build ./...` succeeds
- [x] 1.2 Add `.github/workflows/test.yml` with a ubuntu/macos/windows matrix running `go vet ./...` and `go test ./...`; verify the workflow is green on the first push

## 2. Key discovery

- [x] 2.1 Implement `internal/keys` lookup order with an injectable home/config-dir; verify table tests for all three OS layouts and the explicit-but-missing `SOPS_AGE_KEY_FILE` case pass
- [x] 2.2 Implement secrets-file and upward `.sops.yaml` resolution; verify tests for flag > env > default and parent-directory lookup pass

## 3. CLI conventions

- [x] 3.1 Implement exit-code handling (125 for own failures) and an error type carrying a fix hint; verify a test asserts exit 125 and the hint text for "no key found"
- [x] 3.2 Add a test helper that fails when any known secret test value appears in stdout or stderr; verify it is used by every command test

## 4. Store

- [x] 4.1 Implement `internal/store` decrypt of a dotenv SOPS file with the found age identity; verify against a fixture encrypted by the `sops` CLI with a committed test-only key
- [x] 4.2 Implement encrypt + atomic write (temp file in same directory, rename); verify the output decrypts with `sops -d` in CI (sops installed in the workflow only for this test)
- [x] 4.3 Map "not a recipient" and MAC failures to actionable errors; verify tests for both messages

## 5. sopsy init

- [x] 5.1 Generate an age key at the OS default location with user-only permissions when missing; verify test checks file mode `0600` on unix and the printed public key
- [x] 5.2 Create `.sops.yaml` with the own recipient when missing, report recipient membership when present; verify tests for both branches
- [x] 5.3 Create an empty encrypted secrets file when missing; verify `sops -d` decrypts it in CI
- [x] 5.4 Verify idempotency: a test runs `init` twice and asserts every step reports `present` and no file mtime changed
