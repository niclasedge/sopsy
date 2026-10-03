# Tasks

## 1. Server and security

- [ ] 1.1 `internal/web` server on `127.0.0.1:0` with middleware chain (host check, session, headers); verify `httptest` cases: foreign Host → 403, no token → 403, token → cookie + redirect without token
- [ ] 1.2 CSRF token per session and POST/content-type enforcement; verify tests for missing token, wrong method and wrong content type leave the file unchanged
- [ ] 1.3 Security headers and no body logging; verify a test asserts CSP, no-store, nosniff, no-referrer on every route and that logged output contains no form field
- [ ] 1.4 Idle shutdown with configurable duration; verify a test with a short duration sees the server exit

## 2. Pages

- [ ] 2.1 Setup page driven by the doctor check list with create buttons and CLI equivalents; verify a test on an empty temp HOME shows three missing steps and completes them
- [ ] 2.2 Keys page: list, set, delete with confirmation, add; verify a test sets a value and asserts no response body contains it
- [ ] 2.3 Recipients page: list, copy own key, add, remove with confirmation and rotation warning; verify tests mirror the CLI guards
- [ ] 2.4 Retrieve page with bash/zsh, PowerShell, cmd and cron examples using the resolved file path; verify a snapshot test

## 3. Command

- [ ] 3.1 `sopsy ui [--port N]` with browser open per OS and printed URL fallback; verify manual check on macOS, Linux and Windows, recorded in the PR
