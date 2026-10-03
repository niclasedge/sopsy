# Tasks

## 1. Write path

- [ ] 1.1 Add hash-at-read and compare-before-rename to `internal/store`; verify a test that modifies the file mid-operation gets exit 125 and an unchanged file
- [ ] 1.2 Reject values containing newlines and invalid key names; verify table tests

## 2. Commands

- [ ] 2.1 `sopsy set KEY` with hidden double-entry prompt on TTY and stdin otherwise, refusing a value argument; verify tests for piped value, trailing-newline strip, empty value and argv value
- [ ] 2.2 `sopsy unset KEY`; verify tests for existing and missing key and `sops -d` compatibility in CI
- [ ] 2.3 `sopsy keys` without decrypting; verify a test runs it with no key available and gets the sorted names
- [ ] 2.4 Verify the no-secret-in-output helper passes for every test in this change
