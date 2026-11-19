# Concurrent subscribes can share an invocation ID and drop the fresh connection

Seen on the 3.3.1 beta (rollo logs 2026-11-19): hub 8ee88ca8 (4 times), hub aaab1e7d (4 times), and hub
c2ff195b twice on 3.2.5/3.3.0, so it predates 3.3.1. It shows up on sites with two chargers, after both the
unauthorized restart and a plain network reconnect.

About 1 s after `signalR: client connected` the library logs `error=unknown invocationID 2 …
completionMessage{InvocationID:"2", Result:true}` and closes the connection. Everything queued from the
replay then fails with `charger is not subscribed`. On 8ee88ca8 at 18:00:58 that was 13 W `Send report err`
lines and EG's `cmd.current_session.set_current 13`, which came back as an error. The client reconnects
about 6 s later.

Cause: philippseith/signalr v0.11.0 (the latest release) generates IDs as
`atomic.AddUint64(&l.lastID, 1); return fmt.Sprint(atomic.LoadUint64(&l.lastID))`. Two concurrent
`Invoke` calls can both read 2. The manager has run each subscribe in its own goroutine since #132, so
after a reconnect the chargers' `SubscribeWithCurrentState` calls overlap. The second completion for ID 2
is unknown to the client, and `loop.handleCompletionMessage` closes the connection on it.

Would close it: make `client.invoke` hold a mutex until its result or timeout arrives, so only one
invocation is in flight at a time. Test it with two concurrent `SubscribeCharger` calls against the test
hub, asserting that no completion arrives with an unknown ID.
