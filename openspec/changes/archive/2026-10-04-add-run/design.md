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

## Implementation notes

- **SIGINT from the terminal is not forwarded.** When sopsy is in the
  foreground process group of its controlling terminal, Ctrl+C has already
  reached the child from the kernel; forwarding it would deliver it twice, and
  many programs treat a second interrupt as "force quit". A SIGINT sent to
  sopsy alone (`kill -INT`, no terminal) is forwarded. SIGTERM and SIGHUP are
  always forwarded.
- `--` is optional: flag parsing stops at the first non-flag argument.
  PowerShell 5.1 strips `--` when calling native commands.
- Process tests use the test binary itself as sopsy (`SOPSY_TEST_MAIN=sopsy`)
  and as the helper child, so they need no separate build.
