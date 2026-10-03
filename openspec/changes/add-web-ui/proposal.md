# Proposal

## Why

The CLI covers everything, but first-time setup and "add a new token" are the
moments people get wrong — and the moments where a value most often lands in a
terminal or an AI agent's context. A local, guided UI lets a person type the
value straight into a browser field that sends it to the encrypted file, as
`simonw/llm-keys-ui` does for LLM keys — with SOPS encryption instead of a
plaintext JSON file and with DNS-rebinding protection it lacks.

## What Changes

- `sopsy ui`: start a local web server on `127.0.0.1`, open the browser with a one-time token.
- Pages: Setup (guided, mirrors `init` + `doctor`), Keys (set/delete/add, never show values), Recipients (list, copy own key, add, remove with warning), Retrieve (example commands per shell).
- Security: Host allowlist, token cookie, CSRF, strict CSP, no-store, idle shutdown, no body logging.

## Capabilities

### New Capabilities
- `web-ui`: the local guided web interface and its security properties.

### Modified Capabilities
<!-- none -->

## Impact

New `internal/web` package with embedded templates and CSS. Reuses store, key
editing, recipients and doctor checks. Depends on `add-key-editing`,
`add-recipients` and `add-doctor`.
