# Proposal

## Why

The main way secrets are consumed is by starting a program with them in its
environment — today `sops exec-env file "cmd"`. That needs `sops` installed and
takes the command as one shell string, which breaks quoting and does not exist
in the same form on Windows.

## What Changes

- `sopsy run [--file F] -- <cmd> [args...]`: decrypt the secrets file and start
  `<cmd>` with every key added to its environment.
- Arguments are passed as an argv list, not a shell string.
- Exit code and signals are passed through; Windows support without `exec`.

## Capabilities

### New Capabilities
- `secret-injection`: starting a command with the decrypted secrets in its environment.

### Modified Capabilities
<!-- none -->

## Impact

New `run` subcommand in `internal/cli`, process handling in `internal/runner`
with a unix and a windows implementation. Depends on `add-init` (key discovery,
store, exit conventions).
