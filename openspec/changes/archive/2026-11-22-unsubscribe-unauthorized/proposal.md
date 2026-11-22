# A refused unsubscribe reconnects too

## Why
- `Unregister` unsubscribes a removed charger on the shared connection. On a connection whose token has
  lapsed (issue 4) Easee refuses it as unauthorized: `Unregister` returns the error, Easee keeps streaming
  that charger, and each of its observations logs `Handle observation ... err: no handler` at warning level
  until something else reconnects (issue 5). It hits multi-charger sites that remove or deselect a charger
  more than ~1 h after the last connect.

## What Changes
- An unsubscribe refused with the Unauthorized hub error restarts the SignalR client when other chargers
  remain registered, within the existing once-per-connection limit. The reconnect subscribes only the
  chargers still registered. The last charger's unsubscribe closes the client as before.
- Unchanged: no restart for another error. The cleanup unsubscribe after an unregister race runs on the
  connection whose refusal of `Unregister`'s own unsubscribe already restarted it, so it is left as is.

## Impact
- Affected specs: `observation-streaming`
- Affected code: `src/internal/signalr/manager.go`
