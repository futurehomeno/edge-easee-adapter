# Warn about a clamped offered current only when it changes

## Why
- `clampToMax` warns `Clamp offered current <N>A to max <M>A` on every
  `cmd.current_session.set_current` above the charger's max current. Energy Guard repeats its target
  current on each balancing tick, so a charger whose live max sat at 13 A while the target was 20 A
  logged the same warning 807 times in 8.5 hours on one hub.

## What Changes
- The clamp warning is logged once per charger for each new requested/limit pair; repeats stay
  silent, and a set-current that is not clamped resets it so the next clamp warns again. A start or
  a phase-mode restart, which resend the already clamped current, do not reset it.
- The current written to Easee is unchanged.

## Impact
- Affected specs: `charging-control`
- Affected code: `src/internal/easee/controller.go`
