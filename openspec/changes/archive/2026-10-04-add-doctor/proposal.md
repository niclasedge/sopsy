# Proposal

## Why

When decryption fails, the cause is almost always local setup: key in the wrong
place, key not a recipient, key file readable by others. SOPS reports all of
these with the same misleading message. One command that checks each link of the
chain and says which one is broken saves the debugging session.

## What Changes

- `sopsy doctor`: run every check, print one line per check with `ok` / `warn` /
  `fail` and a fix hint, exit non-zero if any check fails.

## Capabilities

### New Capabilities
- `diagnostics`: the `sopsy doctor` checks and their reporting.

### Modified Capabilities
<!-- none -->

## Impact

New subcommand reusing key discovery, store and config reading. Depends on
`add-init`. The web UI setup page shows the same checks.
