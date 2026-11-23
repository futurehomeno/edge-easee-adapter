# A slow session resumed by a start without a mode stays slow

## Why
- Energy guard pauses a charger with `cmd.charge.stop` and resumes it with a `cmd.charge.start`
  that carries no mode. Since `start-floor-every-start` such a start is raised to
  `initial_charging_current` (16 A), so a session the user started in slow mode comes back at 16 A
  (issue 37).
- The operator decided on 2026-11-23: a slow session that energy guard pauses and resumes comes back
  at its slow current, so the adapter remembers the mode.

## What Changes
- The cache keeps the last explicit start mode of the charger's session. A start with a mode
  replaces it; a start without a mode is treated as that mode.
- The session-finished observation clears it, in the same place that clears the cached requested
  offered current.
- Like the rest of the cache it lives in memory: after an adapter restart a start without a mode is
  not slow (issue 38).

## Impact
- Affected specs: `charging-control`
- Affected code: `src/internal/cache/cache.go`, `src/internal/easee/controller.go`
  (`StartChargepointCharging`), `src/internal/signalr/handler.go` (session-finished clear)
