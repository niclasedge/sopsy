# Design

## Context

Builds on the store from `add-init`. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Drop-in replacement for `sops exec-env` semantics, including override order.
- Same behavior on all three operating systems.

**Non-Goals:**
- Filtering which keys a command receives (later, per-app allowlist).
- Writing the decrypted values anywhere (no `export`, no temp `.env`).

## Decisions

- **Unix: `syscall.Exec` vs. child process.** Use a child process on every OS.
  `exec` would be simpler on unix but leaves two code paths with different signal
  and exit behavior; one model is easier to test. sopsy forwards SIGINT/SIGTERM/
  SIGHUP and exits with the child's code (or 128+signal).
- **Windows:** child started via `os/exec`; Ctrl+C is delivered to the whole
  console process group, so sopsy ignores it itself and waits for the child.
- **Order: decrypt, then resolve `<cmd>`.** Decrypt first (fail 125), then
  `exec.LookPath` (127/126), so a missing command does not hide a broken key
  setup.
- **Env merge:** start from `os.Environ()`, overwrite by key; on Windows match
  names case-insensitively, as the OS does.

## Risks / Trade-offs

- [Values visible in `/proc/<pid>/environ` / `ps eww` to the same user] → inherent
  to environment injection, documented in `docs/design.md` security model.
- [Child ignores the forwarded signal] → sopsy waits; a second Ctrl+C is forwarded
  again, never escalated to kill by sopsy.
