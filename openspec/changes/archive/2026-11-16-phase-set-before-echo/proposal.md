# Send a phase mode requested before the previous one is observed

## Why
- `SetChargepointPhaseMode` skips the Easee call when the target equals the cached internal phase
  mode. Only the SignalR phase-mode observation writes that value, so a second request landing
  before the charger echoes the first is compared with the old mode, treated as a no-op and
  reported as applied while nothing is sent.
- Field log, IT grid, 3 phases: a user's `cmd.phase_mode.set L1L2L3` logs `Set phase mode to auto
  (2)`; energy guard answers with `L3L1` a second later and no `Set phase mode to single phase (1)`
  follows, yet the command succeeds. The charger stays in auto and switches itself to `L1L2L3` the
  next day. The same happened with `L1L2` in the same second as an auto request. Energy guard
  answers a three-phase report on IT within a second, so a user picking three-phase is enough.

## What Changes
- After a successful set-phase-mode call the target replaces the cached internal mode, keeping the
  timestamp of the observation it replaces, so the next request compares against the mode just sent
  and any newer observation still replaces it.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/easee/controller.go`
