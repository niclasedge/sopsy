# web-ui Specification

## Purpose
A local, guided browser interface for setting up sopsy and setting secret values, reachable only from the same machine and never displaying a value.

## Requirements

### Requirement: Local-only start with one-time token
`sopsy ui` SHALL listen on `127.0.0.1` on a free port, generate a random token of at least 128 bits per start, print the URL containing the token, and open it in the default browser on macOS, Linux and Windows.

#### Scenario: Start
- **WHEN** `sopsy ui` runs
- **THEN** the server listens only on the loopback interface and the browser opens the tokenized URL

#### Scenario: No browser available
- **WHEN** opening the browser fails
- **THEN** the URL is still printed and the server keeps running

### Requirement: Request authentication
The server MUST reject with 403 any request whose Host header is not `127.0.0.1:<port>` or `localhost:<port>`, and any request without a valid session; a valid token in the URL SHALL be exchanged for an `HttpOnly`, `SameSite=Strict` session cookie and removed from the URL by redirect.

#### Scenario: DNS rebinding
- **WHEN** a request arrives with `Host: attacker.example:<port>`
- **THEN** the server responds 403

#### Scenario: Missing token
- **WHEN** a request arrives without session cookie or valid token
- **THEN** the server responds 403

### Requirement: CSRF protection
Every state-changing request MUST carry a per-session CSRF token and use POST with `application/x-www-form-urlencoded`; anything else SHALL be rejected without changes.

#### Scenario: Cross-site form post
- **WHEN** a POST arrives without the CSRF token
- **THEN** the server responds 403 and the secrets file is unchanged

### Requirement: Values are write-only
No HTTP response MUST contain a stored secret value or a submitted value; value inputs SHALL be password fields with autocomplete disabled.

#### Scenario: After setting a value
- **WHEN** a value is set through the Keys page
- **THEN** the following page shows the key as set and contains neither the value nor any part of it

### Requirement: Guided setup
While any `sopsy doctor` check fails, the UI SHALL show the Setup page with each step's status, a button to perform the missing `init` step, and the equivalent CLI command for each step.

#### Scenario: Fresh machine
- **WHEN** the UI starts on a machine without key, config and secrets file
- **THEN** the Setup page shows three missing steps, each with a create button and its CLI command

#### Scenario: All checks pass
- **WHEN** every check is ok
- **THEN** the UI opens on the Keys page

### Requirement: Key and recipient pages
The UI SHALL provide the operations of `set`, `unset`, `keys`, `pubkey` and `recipients add/remove` with the same validation, guards and warnings as the CLI, and a Retrieve page with example commands for bash/zsh, PowerShell, cmd and cron.

#### Scenario: Removing a recipient
- **WHEN** a recipient is removed in the UI
- **THEN** the UI asks for confirmation and afterwards shows the rotation warning with all key names

### Requirement: Hardened responses and lifetime
Every response SHALL carry `Cache-Control: no-store`, a Content-Security-Policy allowing only same-origin resources and no inline scripts, `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`; request bodies MUST NOT be logged; the server SHALL shut down after 15 minutes without requests.

#### Scenario: Idle shutdown
- **WHEN** no request arrives for 15 minutes
- **THEN** the server stops and the process exits 0 with a message
