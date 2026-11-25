# 52: a failed login marks a working session not configured

## What happened

Found by the `spec-coverage` change of 2026-11-25, read from the code, not seen on a hub.

`application.Login` (`src/internal/app/app.go`) calls `lifecycle.MarkNotConfigured()` whenever
`Authenticator.Login` fails, a refused password and blank credentials (`ErrEmptyCredentials`) included. The
authenticator stores nothing on a failure, so the stored session of the previous login stays on disk and the
SignalR client keeps running. Still the app now reports not configured, not authenticated and disconnected, and
the reporting and token-refresh tasks, gated on that state, stop. After the next restart `Initialize` finds the
stored credentials and the app is authenticated again.

`TestApplication_Login` ("if Easee API returned an error …") starts from a not-configured app only, so it does not
show which state a running app should keep.

## Reproduce

1. Log in successfully; the app is running and authenticated.
2. Send `cmd.auth.login` with a wrong password (or a blank one).
3. `cmd.app.get_state`: not configured; the chargers keep reporting over SignalR, the periodic tasks stop.
4. Restart the adapter: authenticated again with the old session.

## What would close it

A decision whether a failed re-login on a running app shall keep the current state (and the stored session) or end
the session for good (then also clear the stored credentials, so a restart does not bring it back). Then a test from
a running app and a requirement in `cloud-authentication`.
