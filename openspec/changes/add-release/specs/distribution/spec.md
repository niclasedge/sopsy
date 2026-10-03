# Spec Delta

## Purpose

How sopsy is built, versioned and distributed for macOS, Linux and Windows, including the license notices its compiled-in dependencies require.

## ADDED Requirements

### Requirement: Release artifacts for six targets
Pushing a tag matching `v*` SHALL publish a GitHub release with archives for darwin, linux and windows, each for amd64 and arm64, plus a SHA-256 checksums file.

#### Scenario: Tag pushed
- **WHEN** tag `v0.1.0` is pushed
- **THEN** the release contains six archives and `checksums.txt`, and each archive's checksum matches

### Requirement: License notices shipped
Every release archive SHALL contain `LICENSE` and a `THIRD_PARTY_LICENSES` file listing every compiled-in module with its license text and, for MPL-2.0 modules, the source location of the exact version used.

#### Scenario: SOPS notice
- **WHEN** a release archive is unpacked
- **THEN** `THIRD_PARTY_LICENSES` names `github.com/getsops/sops/v3`, its version, MPL-2.0 and a link to that version's source

### Requirement: Version reporting
`sopsy version` SHALL print the release version, commit hash and build date; a build without release metadata SHALL print `dev`.

#### Scenario: Release build
- **WHEN** `sopsy version` runs from a `v0.1.0` release binary
- **THEN** it prints `0.1.0` with the commit and date

#### Scenario: Local build
- **WHEN** `sopsy version` runs from `go build`
- **THEN** it prints `dev`
