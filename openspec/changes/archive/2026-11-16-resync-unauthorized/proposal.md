# A re-sync refused as unauthorized reconnects once

## Why
- 3.2.5 beta logs on 19 hubs show `Failed to subscribe charger '<id>': ... HubException: Unauthorized
  for <id>`, mostly on the stale-state re-sync (#187, #202). The re-sync re-subscribes a charger on the
  live SignalR connection: it succeeds early in a connection's life, is refused later, and works again
  only after the next reconnect, 1-9 hours later. Observations keep flowing meanwhile, so the only loss
  is that the stale-state repair does nothing for most of each connection's life.
- The SignalR connection carries the access token it was opened with, and nothing renews it while the
  connection lives (see design.md).

## What Changes
- A subscribe on an established connection refused with the Unauthorized hub error, for a charger
  already subscribed on that connection, restarts the SignalR client once, so it reconnects with the
  current access token. The restart reuses Unregister's close-then-start path.
- At most one such restart per connection, never for a charger not yet subscribed on the connection
  (a refusal right after connect, or a charger that left the account), never for another error.

## Impact
- Affected specs: `observation-streaming`
- Affected code: `src/internal/signalr/manager.go`
