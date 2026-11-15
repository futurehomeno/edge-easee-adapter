# Easee access token lifetime and the "24h access token" comment

The comment on `reloginLead` in `src/internal/api/authenticator.go` calls the access token a 24h
token, and the login fixture in `http_test.go` uses `expiresIn: 86400`. Nothing in the code reads the
real lifetime Easee issues; cliffhanger only refreshes 5 minutes before expiry.

The 3.2.5 beta logs (resync-unauthorized) look more like a shorter lifetime: on one hub a re-sync was
refused about 66-68 minutes after a fresh connect, twice. If Easee issues ~1h tokens, the refused
re-subscribe will restart the SignalR client about once an hour while a charger needs re-syncs.

Open:
- What `expiresIn` does Easee return today? A debug line with the parsed expiry on refresh would
  settle it from field logs.
- Does Easee check the token on each hub invocation (the inferred cause), or something else?
- Correct the `reloginLead` comment once the lifetime is known.
