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

## Implementation notes

- **Single-use token** (deviates from "the token stays valid for the server's
  lifetime"): the URL also appears in the process arguments of `open`,
  `xdg-open` or the browser, which other local users can read with `ps`. After
  the first exchange the token is dead; the browser that used it keeps its
  session cookie, so reopening `http://127.0.0.1:<port>/` still works. A new
  link means restarting `sopsy ui`.
- **Cookie:** `HttpOnly; SameSite=Strict`, no `Secure` (plain HTTP on
  loopback); the name carries the port so two `sopsy ui` runs do not overwrite
  each other's cookie. Session value and CSRF token are separate 128-bit values.
- **Middleware order:** headers → log → Host → session → routes, so 403s carry
  the hardening headers too. The token redirect always goes to `/` (the request
  path could be `//host`, an open redirect).
- **No echo of wrong input:** invalid key names and invalid recipients get a
  fixed message without the input — a value or a private key pasted into the
  wrong field must not come back. The CLI keeps quoting the input, since it
  came from the user's own argv.
- **Shared logic:** recipient add/remove moved from `internal/cli` to
  `internal/recipients` and the error → hint mapping to `internal/hint`, so the
  UI applies exactly the CLI's guards and messages.
- **Setup page:** besides one button per step (disabled until its
  prerequisite exists) there is "create everything missing", the UI form of
  `sopsy init`.
- **Retrieve examples** use only a key name that passes `ValidateName` (the
  file is not trusted to hold names that are safe in shell quotes) and escape
  `%` for crontab.
- **Idle timer** resets on authenticated requests only, so unauthenticated
  probing cannot keep the server alive; `--idle` sets the duration.
- **Access log:** method, escaped path, status — no query string, no body.
- **Browser open** uses per-OS files with build tags (`browser_darwin.go`,
  `browser_windows.go`, `browser_other.go`).
