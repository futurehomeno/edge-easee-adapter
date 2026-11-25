# Read the state over REST instead of re-subscribing

Raised 2026-11-25 while closing issue 15 (`charging-watchdog-window`).

The source of the unauthorized restarts (issues 4, 14, 27) is that a state check is a new request on the SignalR
connection, which keeps the token it was opened with. That token ends about an hour after the login, the request is
refused, and the only way to give the connection the new token is to close and reopen it. The 20-minute watchdog makes
such requests rare; the check 15 s after a start or stop still makes them.

A state check over REST (`GET /api/chargers/{id}/state`, named as the alternative in
`resync-state-after-charge-command` design.md) uses the current token on every call, so it is never refused for an old
token and the connection is never reopened for it. The watchdog could then go back to 5 minutes.

Unknown:
- Easee's page for that request and for its rate limits could not be opened on 2026-11-25 (404 on the guessed
  addresses); the fields it returns and the allowed call rate are unverified.
- Whether the values must go through the observation handlers and their timestamp guard, or a plain comparison with
  the cache (re-subscribe only when they differ) is enough.

What would close it: the request and its limits read from Easee's documentation (saved under `openspec/docs`), then a
change that moves the watchdog and the after-command check to it, or a recorded decision not to.
