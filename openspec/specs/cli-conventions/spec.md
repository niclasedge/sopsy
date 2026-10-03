# cli-conventions Specification

## Purpose
Shared behavior of every sopsy command: how failures are reported and the guarantee that no secret value is ever printed.

## Requirements

### Requirement: No secret value in any output
sopsy MUST NOT write a decrypted secret value to stdout, stderr, log output or any error message, including messages about malformed input.

#### Scenario: Malformed line in decrypted content
- **WHEN** a decrypted dotenv line cannot be parsed
- **THEN** the error names the file and line number but not the line content

### Requirement: Own failures exit with 125
When sopsy itself fails (missing key, decryption failure, invalid arguments, I/O error) it SHALL exit with code 125.

#### Scenario: No key available
- **WHEN** no age key exists in any lookup location
- **THEN** sopsy exits 125

### Requirement: Actionable errors without fallback
Every failure message SHALL state what went wrong and the next command or step that fixes it, and sopsy MUST NOT continue with defaults or an empty environment after a failed decryption.

#### Scenario: No key found
- **WHEN** no age key exists in any lookup location
- **THEN** the message lists every searched path and suggests `sopsy init`

#### Scenario: Key is not a recipient
- **WHEN** a key is found but it is not a recipient of the file
- **THEN** the message shows the key's public part and suggests sending `sopsy pubkey` output to someone who can already decrypt

#### Scenario: Tampered file
- **WHEN** the file's MAC does not verify
- **THEN** sopsy exits 125 with a message that the file was modified outside SOPS
