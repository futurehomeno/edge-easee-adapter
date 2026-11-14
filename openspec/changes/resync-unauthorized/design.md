# Design

## Inferred cause
The refusals start some time into a connection's life and stop at the next reconnect. On one hub the
first refusal drifts about 3 minutes a day, close to the drift of the 30-minute token refresh task
(its interval carries a random 0-5 s added once at start). The inference: the access token the
connection was opened with expires or is rotated, and Easee's hub checks the token on each
invocation. This is inferred from log timing and was not confirmed on a live hub.

## What the code confirms
- `client.getConnection` reads the token once per connection and bakes it into the
  `Authorization` header of `signalr.NewHTTPConnection`. Every dial calls it again, so a new
  connection always carries the current token.
- The refresh task (`app.RefreshToken` -> `Ping` -> `Authenticator.AccessToken`) renews the token
  only within cliffhanger's 5-minute `RefreshLead` before expiry, and never touches the SignalR
  client. Nothing renews or reconnects a live connection when its token changes or expires.
- `Connected()` stays true and observations keep flowing on a connection whose token is no longer
  valid, so only invocations (subscribe, re-sync, unsubscribe) see the refusal.

## Decision
Restart the client when an established connection refuses a re-subscribe as unauthorized. The
restart is `restartClient`, the close-then-start-if-chargers-remain path Unregister already used
(extracted from it, not duplicated). Its `startEpoch` snapshot is dropped: after a close decided on
an empty map, a non-empty map can only come from a Register, so the charger count alone says the same.

Rejected: renewing the connection from the token-refresh hook. The hook lives in the app package and
knows nothing about the SignalR client, it would reconnect every charger on every token renewal even
when nothing was refused, and it rests entirely on the unconfirmed cause. Reacting to the refusal
itself is narrower and fixes the observed symptom whatever invalidated the token.

## Loop guard
- Only for a charger already subscribed on the same connection: the subscribe captured
  `isSubscribed` and the connection epoch did not change while it ran. A refusal of a first
  subscribe (the at-connect variant, or a charger that left the account) never restarts, so an
  Unauthorized that persists after the reconnect cannot restart again: the charger is not subscribed
  on the new connection.
- At most one restart per connection: `reconnecting` is set when the restart is decided and cleared
  only by the next connect.
- Only the `HubException: Unauthorized` error; timeouts and other errors keep the existing retry.
- The refused re-sync keeps its normal retry; the reconnect sweep re-subscribes every charger with
  `SubscribeWithCurrentState`, which replays the state the re-sync wanted.

## Unexplained variant
A subscribe refused right after connect that succeeds on the same connection 1-2 minutes later is
untouched: the charger is not subscribed on that connection, so it keeps the backoff retry it has
today. A restart can at worst open a new connection that shows this variant, which the same retry
covers.
