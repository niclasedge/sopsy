# Tasks

## 1. Config editing

- [x] 1.1 Implement `internal/sopsconfig` read/modify/write on `yaml.Node` for the age list of the matching rule; verify golden-file tests that comments and anchors survive add and remove

## 2. Re-encryption

- [x] 2.1 Implement data-key re-encryption to a new recipient set in `internal/store`; verify a test where a second test key can decrypt after add and cannot after remove

## 3. Commands

- [x] 3.1 `sopsy pubkey`; verify a test asserts a single `age1` line
- [x] 3.2 `sopsy recipients` listing with own-key marker, without decrypting; verify test with no private key available
- [x] 3.3 `sopsy recipients add` with validation and already-present handling; verify tests and `sops -d` with the added key in CI
- [x] 3.4 `sopsy recipients remove` with rotation warning and last-recipient guard; verify tests for warning text, key-name list and refusal
