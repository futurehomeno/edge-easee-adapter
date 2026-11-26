# The unauthorized restart can race a logout

Found by the review bot on `resync-unauthorized` (PR #212, P1).

`Logout` closes the SignalR client directly and leaves the chargers registered. The restart is a
`Close()` followed by a `Start()`, so a logout that lands while the restart is closing, or just before
its `Start()`, gets the client started again after it. When the credentials are already cleared the
client only waits for a login, as it does today. When its dial read the token before the clear, the
connection stays up with the old token until the next login or process restart.

The window is a few milliseconds of a restart that happens at most once per connection, and a
`Register` racing a logout starts the client the same way today. Not fixed in PR #212.

Open: is that acceptable, or should the restart move into the client (a `Restart()` that a later
`Close()` cancels, like the existing `startRequested`), which would also remove the manager's
close-then-start path?

## Status 2026-11-25

PR #217 deleted this file instead of moving it to the archive; restored here from the commit before
it. PR #217 ("restart without racing a close") made the fix the open point asks for: the manager calls
`client.Restart()`, which defers its own start so a later `Close()` cancels it
(`src/internal/signalr/client.go:164-205`, `src/internal/signalr/manager.go:215,386`).
