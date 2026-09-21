# pacing-slot-and-teardown

## Why

Three defects in the per-charger pacing channel added by `pace-offered-current-writes`
(#160), filed as [#161](https://github.com/futurehomeno/edge-easee-adapter/issues/161),
[#162](https://github.com/futurehomeno/edge-easee-adapter/issues/162) and
[#163](https://github.com/futurehomeno/edge-easee-adapter/issues/163).

- **A later `set_current` cancels a queued stop.** `pendingDiffers` deliberately refuses to let a
  same-value write displace a stored stop, but `dispatch` replaces `c.pending` unconditionally, so
  a write with a *different* current does exactly what the same-value guard was written to prevent.
  A balancing refresh landing inside the 30s window after `cmd.charge.stop` leaves the charger
  running — including after an emergency pause.
- **The deferred timer outlives the thing.** `dispatch` arms `clock.AfterFunc` and discards the
  handle; `Disconnect` only unregisters the SignalR handler. After `cmd.thing.delete` the timer
  still fires `StopCharging` / `UpdateDynamicCurrent` against a charger the hub no longer owns.
- **`cmd.charge.start` fails after a restart.** The cache is seeded with grid type and phase mode
  but not with max current, so both `RequestedOfferedCurrent` and `MaxCurrent` read 0 and Start
  returns `invalid start current` before any HTTP call. `setOfferedCurrent` already falls back to
  32A for exactly this case; Start does not, so the user cannot start until observation 47 arrives
  — including while SignalR is down and the REST API still works.

## What Changes

- **A stored stop occupies the slot against any later current write.** A stop is a safety command;
  a current write must not displace one, whatever value it carries or whichever command issued it.
  The write is dropped and the stop goes out at its deadline. This supersedes the "latest wins
  across kinds" rule, under which an explicit `cmd.charge.start` behind a stored stop cancelled it:
  nothing on this path tells an operator's start from a balancer's resume, so the stop wins over
  both rather than the adapter deciding a paused charger is safe to resume.
- **The deferred timer is cancelled on teardown.** `dispatch` keeps the `clock.Timer`, and the
  controller's teardown stops it and clears `pending`, so nothing is sent for a deleted thing.
- **Start falls back to the stored supported max current, then to 32A.** The factory seeds the
  cache's max current from the persisted thing state, and Start uses the same clamp
  `setOfferedCurrent` already applies rather than failing on an unseeded cache.

## Impact

- `src/internal/easee/controller.go` — slot policy, timer handle, start fallback
- `src/internal/easee/thing.go` — seed max current into the cache
- `src/internal/easee/connector.go` — cancel on `Disconnect`
