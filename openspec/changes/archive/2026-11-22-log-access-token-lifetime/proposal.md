# Log the access token lifetime

## Why
Issue 4: the `reloginLead` comment calls the Easee access token a 24h token, but nothing reads the
lifetime Easee actually issues. On a hub running 3.3.1 SignalR re-subscribes are refused as
"Unauthorized" roughly once an hour, which looks more like a ~1h token. The field logs cannot
settle it today, because neither the login nor the refresh says what lifetime came back.

## What Changes
- Every login and every token refresh logs, at info level, the `expiresIn` Easee returned and the
  access token expiry the adapter derives from it (the JWT `exp` claim, falling back to now plus
  `expiresIn`), as `[auth] Access token expiresIn=<n>s expiresAt=<RFC3339 local time>`.
- Info, not debug: hubs ship with `log_level: info`, so beta logs answer issue 4 without a level
  change. It costs about one line per token lifetime.
- The line is written by the HTTP client right after the token response is parsed, so the user
  login, the re-login with the stored password and the refresh exchange all log it.
- `reloginLead`, its comment and the refresh timing stay as they are until the lifetime is known.

## Impact
- Affected specs: `cloud-authentication`
- Affected code: `src/internal/api/http.go`, `src/internal/api/authenticator.go` (the login's
  access token expiry moves into `accessTokenExpiry`, shared with the log line)
- Log-only. No behaviour, API call or stored value changes.
