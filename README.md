# sopsy

Set and use secrets from a [SOPS](https://github.com/getsops/sops) + [age](https://age-encryption.org/)
encrypted dotenv file — one binary for macOS, Linux and Windows, with a guided local web UI.

> **Status: design phase.** Nothing is implemented yet. See [`docs/design.md`](docs/design.md)
> and the [issues](https://github.com/niclasedge/sopsy/issues).

## Planned usage

```bash
sopsy init                         # create age key, .sops.yaml and secrets.env (only what is missing)
sopsy ui                           # guided web UI on 127.0.0.1 — set values without a terminal
echo -n "$TOKEN" | sopsy set GITHUB_TOKEN
sopsy keys                         # names only, never values
sopsy run -- ./deploy.sh           # run a command with the secrets in its environment
```

PowerShell:

```powershell
sopsy run -- python app.py
```

- No `sops` or `age` install needed — both are compiled in.
- Files stay fully SOPS-compatible; the regular `sops` CLI keeps working.
- sopsy never prints a secret value.

## License

[Apache-2.0](LICENSE). Release binaries include SOPS (MPL-2.0) and age (BSD-3-Clause);
see `THIRD_PARTY_LICENSES` in each release.
