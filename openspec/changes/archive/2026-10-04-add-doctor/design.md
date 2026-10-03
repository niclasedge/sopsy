# Design

## Context

All checks reuse existing packages; this change adds only orchestration and output.

## Goals / Non-Goals

**Goals:**
- Same check list usable by the CLI and the web UI setup page.

**Non-Goals:**
- Fixing problems automatically (doctor reports, `init` creates).
- Windows ACL inspection — Go has no portable API; the check reports `ok` with a note that the profile ACL applies.

## Decisions

- **Checks as data:** a slice of `Check{Name, Run func() Result}` returning
  status, detail and fix hint; CLI prints them, the UI renders them. Later checks
  that depend on a failed earlier one report `skipped` instead of a second error.
- **Exit code 1 on any `fail`** (not 125: doctor itself worked, it found a problem);
  `warn` keeps exit 0.
- **External tools:** `exec.LookPath("sops")`/`("age")` plus version output —
  informational only, sopsy never uses them.

## Risks / Trade-offs

- [Exit 1 vs. the 125 convention] → documented: 125 means sopsy could not run
  its checks at all, 1 means a check failed.

## Implementation notes

- Statuses are `ok`, `warn`, `fail`, `skipped` plus `info` for the external
  tool lines, so "never `fail`" is visible in the data and not just a
  convention of the printer.
- The recipient check uses the secrets file's recipients when the file
  exists (that is what decryption needs) and falls back to the `.sops.yaml`
  rule before the first `sopsy set`.
- The secrets file is loaded once and shared by the recipient, file and
  decrypt checks.
- `sops --version` runs with `SOPS_DISABLE_VERSION_CHECK=1` and a 3 s timeout
  so doctor never waits on the network.
- Windows reports the key permission check as `ok` with a note that the
  profile ACL applies (no portable ACL API, as decided above).
