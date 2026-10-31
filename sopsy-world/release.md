# Release

## Auf einen Blick

goreleaser builds the archives locally as a dry run and in CI on a pushed
`v*` tag. Pushing a tag publishes a release — only with explicit approval.

## Dry run

`goreleaser release --snapshot --clean` — one archive per target in `dist/`,
each with `LICENSE` and `THIRD_PARTY_LICENSES`.

Wo: `.goreleaser.yaml`, `build/`

## Tagging

Tag a green `main` commit: `git tag vX.Y.Z && git push origin vX.Y.Z`.
Tags with a suffix (`v0.1.0-rc.1`) become GitHub pre-releases.
`release.yml` runs goreleaser; check the assets against `checksums.txt`.

Belege: #7 · #17

Wo: `.github/workflows/release.yml`

## Version stamping

Version, commit and date reach `sopsy version` via `-ldflags -X` on
`internal/cli`; plain `go build` prints `dev`.

Wo: `.goreleaser.yaml`, `internal/cli/`
