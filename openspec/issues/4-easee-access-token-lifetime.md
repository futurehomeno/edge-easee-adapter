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

3.3.1 beta (rollo logs 2026-11-19), hub ac369a68: during charging, the first refusal came 51-68 min after the
previous one. Once it came only 27 min after a network reconnect (03:12:44 -> 03:39:33). That fits a ~1 h
token that was already ~30 min old when the reconnect picked it up, not the age of the connection. Hub
8ee88ca8 16:47:55-16:50:10: on one connection with one token, EH152843 was accepted at once. EUFLF683 was
refused for 2 min and then accepted, with no login, refresh or reconnect in between. So the refusal of
that one charger on the fresh connection is Easee's doing, not a missing login.
