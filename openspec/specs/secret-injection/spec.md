# secret-injection Specification

## Purpose
`sopsy run` starts a command with every key of the decrypted secrets file in its environment, as a cross-platform replacement for `sops exec-env`.

## Requirements

### Requirement: Inject all keys into the child environment
`sopsy run -- <cmd> [args...]` SHALL start `<cmd>` with the current environment plus every key of the decrypted secrets file, and values from the file SHALL override variables of the same name already present.

#### Scenario: Value reaches the child
- **WHEN** the file contains `API_TOKEN` and `sopsy run -- printenv API_TOKEN` runs
- **THEN** the child prints the decrypted value and sopsy itself prints nothing else

#### Scenario: File overrides existing variable
- **WHEN** `API_TOKEN=old` is set in the calling environment and the file contains another value
- **THEN** the child sees the value from the file

### Requirement: Arguments are passed verbatim
sopsy SHALL pass everything after `--` to the child as separate arguments without shell interpretation.

#### Scenario: Argument with spaces and quotes
- **WHEN** `sopsy run -- printf '%s\n' "a b" "it's"` runs
- **THEN** the child receives exactly two arguments after the format string, `a b` and `it's`

### Requirement: Exit code pass-through
sopsy run SHALL exit with the child's exit code, with 126 when the command is not executable, with 127 when it is not found, and with 125 when sopsy fails before starting the child.

#### Scenario: Child fails
- **WHEN** the child exits with code 3
- **THEN** sopsy exits with code 3

#### Scenario: Command not found
- **WHEN** the command does not exist
- **THEN** sopsy exits with code 127

#### Scenario: Decryption fails
- **WHEN** the secrets file cannot be decrypted
- **THEN** the child is not started and sopsy exits 125

### Requirement: Interrupts reach the child
sopsy run SHALL forward interrupt and termination signals to the child on macOS and Linux, and Ctrl+C on Windows, and SHALL wait for the child to exit before exiting itself.

#### Scenario: Ctrl+C during a long-running child
- **WHEN** the user presses Ctrl+C while the child runs
- **THEN** the child receives the interrupt and sopsy exits with the child's resulting code
