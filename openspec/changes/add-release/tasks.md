# Tasks

## 1. Version

- [x] 1.1 Add `sopsy version` with ldflags-injected version/commit/date and `dev` fallback; verify a test for the fallback and a `go build -ldflags` test for injected values

## 2. Release pipeline

- [x] 2.1 Add `.goreleaser.yaml` for six targets with checksums and a `go-licenses` before-hook; verify `goreleaser release --snapshot --clean` locally produces six archives each containing `LICENSE` and `THIRD_PARTY_LICENSES`
- [ ] 2.2 Add `.github/workflows/release.yml` triggered on `v*` tags; verify by pushing `v0.1.0-rc.1` and checking the release assets and checksums — open: needs the maintainer to push the tag, tracked in #17

## 3. Docs

- [ ] 3.1 README install section (download, `go install`, Gatekeeper/SmartScreen handling); verify the commands by running them on a clean macOS and Windows machine — open: tracked in #17
