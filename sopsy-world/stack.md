# Software stack

## Auf einen Blick

Go with the standard library first: SOPS and age as libraries, `flag` for the
CLI, `net/http` + `html/template` for the web UI, no JS build step. Lint,
CI and release are configured in the repo.

## Libraries

| Area | Choice |
|---|---|
| Language | Go (version in `go.mod`), `CGO_ENABLED=0` for releases |
| SOPS | `github.com/getsops/sops/v3` as a library (dotenv store, aes, config, keyservice types) |
| age | `filippo.io/age` |
| Hidden prompt | `golang.org/x/term` |
| `.sops.yaml` edits | `go.yaml.in/yaml/v3` on `yaml.Node` (keeps comments and anchors) |
| CLI | standard library `flag` with a small dispatcher — no cobra |
| Web UI | `net/http` + `html/template` + `go:embed`; no JS framework, no build step |

Wo: `go.mod`, `internal/web/templates/`, `internal/web/static/`

## Tooling and CI

| Area | Choice |
|---|---|
| Lint | `golangci-lint` v2 (`.golangci.yml`) |
| CI | GitHub Actions: `test.yml` (OS matrix ubuntu, macos, windows + lint), `release.yml` (goreleaser on `v*` tags) |
| Release | goreleaser, targets in `.goreleaser.yaml`, checksums, `THIRD_PARTY_LICENSES` via `go-licenses` |

GitHub Actions are pinned to a commit SHA (with the version as a comment);
downloaded tools are checksum-verified. Workflows use least-privilege
`permissions`.

Wo: `.golangci.yml`, `.github/workflows/test.yml`, `.github/workflows/release.yml`, `.goreleaser.yaml`

## Dependencies

Add a dependency only when the standard library cannot reasonably do the job,
and check its license (it ends up in `THIRD_PARTY_LICENSES`).

Wo: `go.mod`, `build/third-party-licenses.sh`
