# Design

## Context

`.sops.yaml` decides the recipients of *new* files; an existing file carries its
own recipient list in its metadata. Both have to change together.

## Goals / Non-Goals

**Goals:**
- `.sops.yaml` and file metadata never disagree after a sopsy command.
- Preserve comments and other rules in `.sops.yaml`.

**Non-Goals:**
- Recipient types other than age (KMS, Vault, PGP) — kept intact if present, not managed.
- Rotating values automatically.

## Decisions

- **Edit `.sops.yaml` via `yaml.Node`** (`gopkg.in/yaml.v3`) to keep comments,
  anchors and ordering; only the age list of the rule matching the secrets file
  is touched. Alternative: regenerate the file — rejected, destroys comments.
- **Re-encryption = SOPS `updatekeys` semantics:** decrypt the data key with the
  own identity, encrypt it to the new recipient set; values are not re-encrypted.
  Removal therefore does not change ciphertexts — hence the rotation warning.
- **Order of writes:** write the secrets file first, then `.sops.yaml`. If the
  second write fails, the error says exactly which file is ahead and how to fix it.

## Risks / Trade-offs

- [Anchors in `.sops.yaml` (`&macbook`)] → keys referenced by alias are resolved
  for listing; adding appends a plain entry, removing an aliased key removes the
  alias reference and leaves the anchor definition, reported in the output.
- [Removed recipient still has old ciphertext in git history] → cannot be fixed
  technically; rotation warning is mandatory output.

## Implementation notes

- `go.yaml.in/yaml/v3` (the maintained successor of `gopkg.in/yaml.v3`,
  already in SOPS' module graph) is used for `yaml.Node` editing. Comments and
  anchors survive; blank lines between top-level blocks do not (a yaml.v3
  limitation), so `.sops.yaml` is only rewritten when its rule changes.
- The secrets file is rewritten like `sops updatekeys`: only the `sops_age__*`
  metadata lines change; value ciphertexts, `sops_lastmodified` and the MAC
  stay byte-identical, which keeps the git diff minimal.
- `add` and `remove` repair disagreement: a key present in only one of
  `.sops.yaml` and the file is added to / removed from the other.
- A `.sops.yaml` rule with several `key_groups`, or an age list that is itself
  an alias (`age: *keys`), is refused with a message instead of edited.
- File writes use `internal/atomicfile`, shared by the store and `.sops.yaml`.
- `.sops.yaml` with CRLF line endings is parsed as LF and written back as
  CRLF: yaml.v3 otherwise invents blank lines around comments (found by the
  Windows CI job on a CRLF checkout).
