# Spec Delta

## Purpose

`sopsy doctor` checks each link from key to decrypted file and names the first broken one with a fix, without ever printing a value.

## ADDED Requirements

### Requirement: Checks in chain order
`sopsy doctor` SHALL run these checks in order and print one line each with status `ok`, `warn` or `fail`: key file found (and which path), key file permissions, `.sops.yaml` found, own key is a recipient, secrets file found, test decryption succeeds (reporting only the number of keys), external `sops`/`age` binaries present (informational, never `fail`).

#### Scenario: Healthy setup
- **WHEN** everything is in place
- **THEN** every check prints `ok` (or an informational line for external tools) and the exit code is 0

#### Scenario: Not a recipient
- **WHEN** the key exists but is not among the file's recipients
- **THEN** that check prints `fail` with the own public key and the hint to share `sopsy pubkey` output, and the exit code is 1

### Requirement: Permission warning
On macOS and Linux, `sopsy doctor` SHALL report `warn` when the key file is readable or writable by group or others, with the exact command to fix it.

#### Scenario: World-readable key
- **WHEN** the key file has mode `0644`
- **THEN** the check prints `warn` and suggests `chmod 600 <path>`

### Requirement: No values
`sopsy doctor` MUST NOT print any decrypted value; the decryption check reports only success and the number of keys.

#### Scenario: Successful test decryption
- **WHEN** the test decryption succeeds on a file with 3 keys
- **THEN** the line reads as ok with `3 keys` and contains no value
