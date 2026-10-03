# Proposal

## Why

Changing one value today means `sops edit` (plaintext in a temp file and an
editor) or `sops set` with a JSON-quoted value. Both are easy to get wrong, and
the value tends to end up in argv, shell history or an AI agent's context.

## What Changes

- `sopsy set KEY`: read the value from a hidden prompt (TTY) or stdin; never from argv.
- `sopsy unset KEY`: remove a key.
- `sopsy keys`: list key names without decrypting.
- Concurrent-change protection for every write.

## Capabilities

### New Capabilities
- `key-editing`: setting, removing and listing keys of the secrets file.

### Modified Capabilities
<!-- none -->

## Impact

New subcommands in `internal/cli`; write path in `internal/store` gains a
compare-before-write check. Depends on `add-init`. The web UI (`add-web-ui`)
reuses this write path.
