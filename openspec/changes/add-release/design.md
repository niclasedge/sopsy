# Design

## Context

Release tooling only; no runtime behavior beyond `sopsy version`.

## Goals / Non-Goals

**Goals:**
- Reproducible, CGO-free builds for six targets from one workflow.
- License compliance for compiled-in dependencies.

**Non-Goals:**
- Apple notarization and Windows code signing (need paid certificates).
- Homebrew tap, winget, Scoop, Linux packages — later; note that the name `sopsy`
  is taken on PyPI/conda-forge by an unrelated Python wrapper, so a package-manager
  name may need a suffix.

## Decisions

- **goreleaser** with `CGO_ENABLED=0`, `-trimpath`, ldflags for version/commit/date.
  Alternative: hand-written matrix workflow — rejected, reinvents archives and checksums.
- **`go-licenses report`** (google/go-licenses) generates `THIRD_PARTY_LICENSES`
  in a `before` hook; the release fails if any dependency has an unknown license.
- **Archive format:** `.tar.gz` for darwin/linux, `.zip` for windows.

## Risks / Trade-offs

- [Unsigned binaries trigger Gatekeeper/SmartScreen] → README documents
  `xattr -d com.apple.quarantine sopsy` and the SmartScreen "Run anyway" path;
  downloads via `curl` are not quarantined.
- [Binary size ~40 MB due to SOPS' cloud SDKs] → accepted for v1 (see `add-init` design).
