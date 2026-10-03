# Tasks

## 1. Checks

- [ ] 1.1 Implement the check list as data with `ok`/`warn`/`fail`/`skipped` results; verify unit tests for each check in isolation
- [ ] 1.2 Implement the unix permission check with a fix command; verify a test with a `0644` key file reports `warn` and `chmod 600 <path>`

## 2. Command

- [ ] 2.1 Wire `sopsy doctor` with exit 0 / 1 semantics; verify tests for a healthy setup, a non-recipient key, and a missing `.sops.yaml` (later checks `skipped`)
- [ ] 2.2 Verify the no-secret-in-output helper passes and the decrypt check prints only the key count
