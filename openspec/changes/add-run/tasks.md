# Tasks

## 1. Runner

- [ ] 1.1 Implement env merge (file overrides, case-insensitive names on Windows); verify unit tests for override and Windows casing
- [ ] 1.2 Implement child start with argv pass-through and exit-code mapping (child code, 126, 127, 125); verify tests using a small helper binary built in the test
- [ ] 1.3 Implement signal forwarding on unix and Ctrl+C handling on Windows; verify a unix test sends SIGINT and asserts the child's handler ran, and a Windows CI job runs the same scenario via `GenerateConsoleCtrlEvent`

## 2. Command

- [ ] 2.1 Wire `sopsy run [--file F] -- <cmd>` with decrypt-before-lookup order; verify test: broken key + missing command exits 125, not 127
- [ ] 2.2 End-to-end in CI on all three OS: `init` → set a test value via the store → `sopsy run -- printenv` (PowerShell `Get-ChildItem env:` on Windows) shows it; verify the no-secret-in-output helper passes for sopsy's own output
