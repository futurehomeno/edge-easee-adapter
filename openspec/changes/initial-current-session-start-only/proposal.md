# The initial charging current is for a session start, not a resume

## Why
- Since PR 98 (3.1) a start in mode `normal`, and since PR 237 (3.4.2) every start that is not slow, raises the cached
  requested current to at least `initial_charging_current` (default 16 A). Energy guard resumes a charger it paused
  with `cmd.charge.start` without a mode, so its resume starts at 16 A instead of the 7 A it had balanced the charger
  to. In hubsim it3-easee the breaker check fails (issue 42).
- On 2026-11-23 the operator decided that the 16 A start stays for every start that is not slow, and issue 42 was
  closed on that decision. On 2026-11-24 the operator reversed it for resumes: "initial current is for session start,
  not resume", "it should not be for resume so fix PR 98".
- The same day the operator added, to match Zaptec: a start after the adapter's own pause (a `set_current` below
  `min_charging_current`) starts at `initial_charging_current`, not below `min_charging_current`, capped by the
  charger maximum.

## What Changes
- A session start is a `cmd.charge.start` before the charger has been seen charging in the session: since the car
  was connected or since the previous session ended. A `set_current` before it does not make it a resume. A session
  start keeps today's current: the cached requested current raised to at least `initial_charging_current`, or the
  charger maximum when no current is cached (PR 98 kept that so installs without a power manager start at full
  current).
- Every later `cmd.charge.start` in the session is a resume: it starts at the cached requested current, the current in
  force before the pause, never raised to `initial_charging_current`. With no current cached it starts as a session
  start would.
- A start while the adapter's own pause is remembered starts at the greater of `initial_charging_current` and
  `min_charging_current`, whether or not the session has charged.
- The charger maximum (32 A when unknown) caps every start, as before. Slow mode, the phase-mode bounce, the pause
  below the minimum and the start's skipped minimum check are unchanged.
- The cache keeps whether the session has charged: a charger state observation of charging sets it, and the
  session-finished observation that clears the cached requested current clears it. It is not kept across an adapter
  restart: the replayed charger state sets it again when the charger is charging.

## Impact
- Affected specs: `charging-control` (Start Charging)
- Affected code: `src/internal/cache` (the session-charged value), `src/internal/signalr/handler.go`
  (`handleChargerState`), `src/internal/easee/controller.go` (`StartChargepointCharging`), README
- After this change energy guard's resume no longer starts above the current it had balanced the charger to; a car
  that only starts at 16 A gets it at the session start, not on a resume.
