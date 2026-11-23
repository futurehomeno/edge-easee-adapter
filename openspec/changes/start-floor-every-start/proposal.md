# Every start that is not slow mode starts at the start current

## Why
- A start without a mode keeps the cached offered current. Energy guard resumes a paused charger
  with such a start, so after a cut to 7 A the charger starts again at 7 A.
- The operator decided on 2026-11-23: a start is at 16 A, limited by the charger maximum, then the
  balancer adjusts, because some cars need the higher current to start.
- The Zaptec adapter already raises the current on every start (`raiseStartCurrent`).

## What Changes
- `cmd.charge.start` raises the cached offered current to at least `initial_charging_current`
  (default 16 A) for every start that is not slow mode, with or without a mode. Until now only mode
  `normal` got this floor.
- The charger's max current still caps the start current, as before.
- Slow mode is unchanged.

## Impact
- Affected specs: `charging-control`
- Affected code: `src/internal/easee/controller.go` (`StartChargepointCharging`)
- After energy guard resumes a charger it offers up to 16 A until the next balancer run lowers it.
