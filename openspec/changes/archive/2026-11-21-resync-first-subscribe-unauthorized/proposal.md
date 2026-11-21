# A first subscribe refused as unauthorized reconnects once per charger

## Why
- The unauthorized restart (`resync-unauthorized`) fires only for a charger already subscribed on the
  connection. A charger registered on a connection whose token has lapsed, or repaired there by
  `repairReplacedSubscription`, is refused on every backoff retry (up to 10 min apart) and stays
  `ChargerNotSubscribed` until something else reconnects: another charger's refused re-sync, a network
  drop or an adapter restart (issue 13). It hits multi-charger sites that add or re-select a charger
  more than ~1 h after the last connect (issue 4).

## What Changes
- A first subscribe refused with the Unauthorized hub error restarts the SignalR client too, at most
  once per charger until that charger next subscribes successfully. The reconnect sweep does not reset
  that guard, so a charger that left the account costs one extra reconnect per adapter run and then
  keeps the backoff retry it has today.
- Unchanged: at most one restart per connection, no restart for another error, no restart when the
  same charger's subscribe is refused again on the new connection. A refused unsubscribe (issue 5)
  stays out of scope.

## Impact
- Affected specs: `observation-streaming`
- Affected code: `src/internal/signalr/manager.go`
- The unexplained refusal right after connect that succeeds 1-2 minutes later on the same connection
  now costs one reconnect, and a connectivity blip for every charger (issue 14).
