# Keep a matching phase echo's timestamp, restart the SignalR client without racing a close

## Why
- Since `phase-set-before-echo` the sent target is cached with the timestamp of the observation it
  replaced. `handlePhaseMode` drops an observation equal to the cached mode before the timestamp
  guard, so the echo of that target never records its own timestamp. The SignalR library runs each
  hub call on its own goroutine, so the echoes of two quick sets can be dispatched out of order, and
  the older one is then accepted over the newer mode.
- The unauthorized restart is the manager's `Close()` then `Start()`. A `Logout` or `Stop` closing
  the client while that `Close()` drains returns at once, and the `Start()` then brings the client
  back up after the caller was told it was closed (issue 6, closed by this change).

## What Changes
- A phase-mode observation equal to the cached mode still goes through the timestamp guard, so its
  timestamp is kept, and still publishes nothing.
- The client gets `Restart()`: it closes a running client and launches it again unless a `Close()`
  arrives before the relaunch. On a client that is not running or already closing it does nothing,
  and leaves a `Start` that close deferred in place. The refused re-subscribe restarts through it;
  `Unregister` keeps its close-then-start path.

## Impact
- Affected specs: `phase-mode`, `observation-streaming`
- Affected code: `src/internal/signalr/handler.go`, `src/internal/signalr/client.go`,
  `src/internal/signalr/manager.go`
