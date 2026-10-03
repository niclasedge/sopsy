# Proposal

## Why

A single binary only helps if people can get it for their platform without a Go
toolchain, and if its licenses are in order: SOPS is MPL-2.0, which requires
binary distributions to point recipients to its source.

## What Changes

- goreleaser configuration producing archives for darwin, linux and windows on amd64 and arm64 on every `v*` tag.
- SHA-256 checksums file per release.
- `THIRD_PARTY_LICENSES` generated from the module graph and shipped in every archive.
- `sopsy version` printing version, commit and build date.
- README install section: release download, `go install`, and how to handle Gatekeeper/SmartScreen warnings for unsigned binaries.

## Capabilities

### New Capabilities
- `distribution`: release artifacts, version reporting and license notices.

### Modified Capabilities
<!-- none -->

## Impact

`.goreleaser.yaml`, `.github/workflows/release.yml`, `version` subcommand with
ldflags, README. Independent of the feature changes except `add-init` (module exists).
