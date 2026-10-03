# Design

## Context

Reuses the write path from `add-key-editing`, recipients from `add-recipients`
and the check list from `add-doctor`. Reference: `simonw/llm-keys-ui` (Starlette,
plaintext JSON store, CSRF token, CSP, no Host validation).

## Goals / Non-Goals

**Goals:**
- Zero build step: templates and CSS embedded with `go:embed`.
- Every security property testable with `httptest`.

**Non-Goals:**
- Remote access (Tailscale, LAN). Loopback only in v1.
- Showing or copying secret values.
- JavaScript frameworks; at most a few lines of same-origin JS for "copy public key".

## Decisions

- **Server:** `net/http` with an explicit middleware chain: host check → session
  check → handler → security headers. Templates via `html/template` (auto-escaping).
- **Token flow:** `GET /?t=<token>` sets the cookie and redirects to `/` so the
  token leaves the address bar and history. The token stays valid for the server's
  lifetime so the printed URL works for re-opening the tab.
- **CSRF:** per-session random token in a hidden field, compared in constant time.
- **Browser open:** `open` (macOS), `xdg-open` (Linux), `rundll32 url.dll,FileProtocolHandler` (Windows).
- **Port:** `127.0.0.1:0`, actual port read from the listener; `--port` to pin one.
- **Idle timer:** reset on every authenticated request; shutdown via `http.Server.Shutdown`.

## Risks / Trade-offs

- [Another local user can connect to the port] → Host check does not help there;
  the token does. 128-bit random token, constant-time compare.
- [Token in printed URL visible in terminal scrollback] → same trust level as the
  key file itself (same user); documented.
- [Browser extensions can read the page] → they never see values after submission;
  during typing, out of scope.
