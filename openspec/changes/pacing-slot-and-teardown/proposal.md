# pacing-slot-and-teardown

## Why

Three defects in the per-charger pacing channel added by `pace-offered-current-writes`
(#160), filed as [#161](https://github.com/futurehomeno/edge-easee-adapter/issues/161),
[#162](https://github.com/futurehomeno/edge-easee-adapter/issues/162) and
[#163](https://github.com/futurehomeno/edge-easee-adapter/issues/163).

- **A phase-mode change can stop charging for good.** The restart dispatches a pause and a resume
  back to back, and the resume is always inside the window the pause just opened, so both compete
  for the channel's single slot. Whichever rule the slot uses, one half is lost: under "the stop
  wins" the resume is dropped and the charger stays paused; under "the latest wins" the pause is
  dropped and the mode never applies. A restart has to be stored as an ordered pair for either
  outcome to be correct.
- **The deferred timer outlives the thing.** `dispatch` arms `clock.AfterFunc` and discards the
  handle; `Disconnect` only unregisters the SignalR handler. After `cmd.thing.delete` the timer
  still fires `StopCharging` / `UpdateDynamicCurrent` against a charger the hub no longer owns.
- **`cmd.charge.start` fails after a restart.** The cache is seeded with grid type and phase mode
  but not with max current, so both `RequestedOfferedCurrent` and `MaxCurrent` read 0 and Start
  returns `invalid start current` before any HTTP call. `setOfferedCurrent` already falls back to
  32A for exactly this case; Start does not, so the user cannot start until observation 47 arrives
  — including while SignalR is down and the REST API still works.

## What Changes

- **The slot always holds the latest command, whatever its kind.** A stop displaces a stored current
  write and a current write displaces a stored stop, the dedup notwithstanding. The charger acts on
  whatever it last heard, so keeping an older command makes the adapter answer a command it did not
  carry out. The cost is that a balancing refresh inside the window now cancels a pause, an emergency
  pause among them — nothing on this path tells one stop from another. See `design.md`.
- **A restart is stored as an ordered pair.** The phase-mode bounce puts the pause and the resume in
  the channel together; the resume goes out one wait time after the pause, and a later command
  discards both halves rather than leaving the charger paused or resuming it at a superseded current.
- **The deferred timer is cancelled on teardown.** `dispatch` keeps the `clock.Timer`, and the
  controller's teardown stops it and clears `pending`, so nothing is sent for a deleted thing.
- **Start falls back to the stored supported max current, then to 32A.** The factory seeds the
  cache's max current from the persisted thing state, and Start uses the same clamp
  `setOfferedCurrent` already applies rather than failing on an unseeded cache.

## Impact

- `src/internal/easee/controller.go` — slot policy, timer handle, start fallback
- `src/internal/easee/thing.go` — seed max current into the cache
- `src/internal/easee/connector.go` — cancel on `Disconnect`
