# Design

## Context

Uses the store from `add-init`. The web UI calls the same write path, so the
guarantees here apply to both.

## Goals / Non-Goals

**Goals:**
- One write path with atomic replace and change detection.
- No value in argv, echo or output.

**Non-Goals:**
- Reading values back (`get`/`view`) — deliberately absent in v1.
- Editing in `$EDITOR`.

## Decisions

- **Hidden prompt:** `golang.org/x/term.ReadPassword`, works on all three OS.
  Two entries must match, like a password change.
- **Trailing newline:** strip exactly one `\n` (or `\r\n`) from piped input so
  `echo "$X" | sopsy set` behaves as expected; preserve everything else.
- **Listing without decrypting:** in the SOPS dotenv format key names are stored
  in plaintext, so `keys` parses names and skips `sops_*` metadata entries.
- **Change detection:** SHA-256 of the file bytes at read time, compared right
  before rename. Alternative: OS file locks — rejected, behave differently on
  Windows and do not protect against editors that replace the file.

## Risks / Trade-offs

- [Race window between hash check and rename] → milliseconds; acceptable for a
  human-driven tool, documented.
- [Values with embedded newlines] → the SOPS dotenv store cannot represent them;
  reject with a clear error instead of corrupting the file.
