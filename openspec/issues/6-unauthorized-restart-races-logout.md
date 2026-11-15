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
