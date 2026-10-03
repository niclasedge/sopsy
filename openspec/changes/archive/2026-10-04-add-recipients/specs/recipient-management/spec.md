# Spec Delta

## Purpose

Shows and changes who can decrypt the secrets file, keeping `.sops.yaml` and the file's encrypted data key in sync in a single step.

## ADDED Requirements

### Requirement: Show own public key
`sopsy pubkey` SHALL print the public key of the age identity found by the key lookup, and nothing else, on one line.

#### Scenario: Key present
- **WHEN** `sopsy pubkey` runs on a machine with a key
- **THEN** a single `age1…` line is printed and the exit code is 0

### Requirement: List recipients
`sopsy recipients` SHALL list the age recipients of the secrets file, marking the own key, without requiring decryption.

#### Scenario: Own key marked
- **WHEN** the own public key is among the recipients
- **THEN** that line is marked as the own key

### Requirement: Add a recipient in one step
`sopsy recipients add <age1…>` SHALL validate the public key, add it to the matching creation rule in `.sops.yaml`, and re-encrypt the file's data key to the new recipient set before reporting success.

#### Scenario: New machine gains access
- **WHEN** a valid public key is added
- **THEN** a machine holding the matching private key can decrypt the file with `sopsy run` and with `sops -d`

#### Scenario: Invalid key
- **WHEN** the argument is not a valid age public key
- **THEN** sopsy exits 125 and changes neither `.sops.yaml` nor the file

#### Scenario: Already a recipient
- **WHEN** the key is already listed
- **THEN** sopsy reports so and changes nothing

### Requirement: Remove a recipient with a rotation warning
`sopsy recipients remove <age1…>` SHALL remove the key from `.sops.yaml`, re-encrypt the data key without it, and print a warning that every value must be rotated, listing the key names, and MUST refuse to remove the last remaining recipient.

#### Scenario: Removal
- **WHEN** a listed recipient is removed
- **THEN** the removed private key can no longer decrypt the new file, and the output lists all key names under a rotation warning

#### Scenario: Last recipient
- **WHEN** the only remaining recipient would be removed
- **THEN** sopsy exits 125 without changes
