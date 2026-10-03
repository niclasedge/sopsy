# Spec Delta

## Purpose

The `sopsy init` command brings a machine and a project from nothing to a working, SOPS-compatible encrypted secrets file, creating only what is missing.

## ADDED Requirements

### Requirement: Create a missing age key
`sopsy init` SHALL generate an age key pair when no key is found by the key lookup, write it to `<user config dir>/sops/age/keys.txt`, and restrict the file to the current user (mode `0600` on macOS/Linux; inside the user profile on Windows).

#### Scenario: Fresh machine
- **WHEN** `sopsy init` runs and no key exists
- **THEN** a key file is created at the OS default location and its public key is printed

#### Scenario: Existing key is kept
- **WHEN** `sopsy init` runs and a key already exists in any lookup location
- **THEN** the key is left unchanged and reported as present

### Requirement: Create a missing .sops.yaml
`sopsy init` SHALL create `.sops.yaml` in the working directory when none is found, with one creation rule matching the secrets file and the own public key as the only age recipient.

#### Scenario: No config yet
- **WHEN** no `.sops.yaml` is found
- **THEN** one is created listing the own public key as recipient

#### Scenario: Existing config is not modified
- **WHEN** a `.sops.yaml` already exists
- **THEN** it is left unchanged, and init reports whether the own key is among its recipients

### Requirement: Create a missing secrets file
`sopsy init` SHALL create an empty encrypted dotenv secrets file when the resolved secrets file does not exist, and the result MUST be decryptable by the `sops` CLI.

#### Scenario: New secrets file
- **WHEN** the secrets file does not exist
- **THEN** an encrypted file with no keys is created and `sops -d` on it succeeds

### Requirement: Idempotent
Running `sopsy init` repeatedly SHALL change nothing once everything exists, and SHALL print one line per step stating `created` or `present`.

#### Scenario: Second run
- **WHEN** `sopsy init` runs twice in a row
- **THEN** the second run reports every step as `present` and modifies no file
