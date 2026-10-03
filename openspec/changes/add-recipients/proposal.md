# Proposal

## Why

Giving a second machine or person access today means editing `.sops.yaml` by
hand and then remembering `sops updatekeys` — until that runs, the new recipient
is still locked out and nothing says so. Removing someone is worse: re-encrypting
is not enough, the values have to be rotated, and no tool says that either.

## What Changes

- `sopsy pubkey`: print the own age public key, the thing a new member sends.
- `sopsy recipients`: list recipients of the secrets file.
- `sopsy recipients add <age1…>`: add to `.sops.yaml` and re-encrypt in one step.
- `sopsy recipients remove <age1…>`: remove, re-encrypt, and print a rotation warning listing every key name.

## Capabilities

### New Capabilities
- `recipient-management`: showing, adding and removing who can decrypt the secrets file.

### Modified Capabilities
<!-- none -->

## Impact

New subcommands; `.sops.yaml` read/modify/write in `internal/sopsconfig`;
re-encryption of the data key in `internal/store`. Depends on `add-init`.
