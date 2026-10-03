# key-editing Specification

## Purpose
Setting, removing and listing individual keys of the encrypted secrets file without the value ever appearing in argv, a terminal echo or command output.

## Requirements

### Requirement: Set a value without argv
`sopsy set KEY` SHALL read the value from a hidden prompt when stdin is a terminal and from stdin otherwise, and MUST NOT accept the value as a command-line argument.

#### Scenario: Interactive entry
- **WHEN** `sopsy set API_TOKEN` runs in a terminal
- **THEN** sopsy prompts without echoing input, asks for confirmation by a second entry, and stores the value

#### Scenario: Piped entry
- **WHEN** a value is piped into `sopsy set API_TOKEN`
- **THEN** the value is stored with at most one trailing newline removed

#### Scenario: Value given as argument
- **WHEN** `sopsy set API_TOKEN secret` is run
- **THEN** sopsy exits 125 without writing and explains that values are read from stdin or the prompt

#### Scenario: Empty value
- **WHEN** the entered value is empty
- **THEN** sopsy exits 125 without writing

### Requirement: Valid key names
sopsy SHALL accept key names matching `[A-Za-z_][A-Za-z0-9_]*` and reject all others before writing.

#### Scenario: Invalid name
- **WHEN** `sopsy set 1BAD-NAME` is run
- **THEN** sopsy exits 125 without writing

### Requirement: Remove a key
`sopsy unset KEY` SHALL remove the key and report whether it existed.

#### Scenario: Key exists
- **WHEN** `sopsy unset API_TOKEN` runs and the key exists
- **THEN** the key is removed and the file still decrypts with `sops -d`

#### Scenario: Key missing
- **WHEN** the key does not exist
- **THEN** sopsy reports that nothing was removed and exits 0

### Requirement: List names without decrypting
`sopsy keys` SHALL print one key name per line, sorted, without requiring a usable private key.

#### Scenario: Machine without key
- **WHEN** `sopsy keys` runs on a machine with no age key
- **THEN** the names are printed and the exit code is 0

### Requirement: Writes keep the file SOPS-compatible and protected against concurrent changes
Every write SHALL keep the existing data key and recipients, recompute the MAC, replace the file atomically, and abort with exit 125 if the file changed on disk since it was read.

#### Scenario: Concurrent modification
- **WHEN** the file is modified by another process between read and write
- **THEN** sopsy does not write and tells the user to retry

#### Scenario: Compatibility
- **WHEN** a value is set with sopsy
- **THEN** `sops -d` decrypts the file and shows the new value, and other keys are unchanged
