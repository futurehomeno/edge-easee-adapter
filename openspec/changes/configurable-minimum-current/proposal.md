# configurable-minimum-current

## Why

Energy managers send `cmd.current_session.set_current` with values under 6 A, 0 A included, to pause a charger.
Cliffhanger rejects any current under 6 A before the controller runs, so on Easee such a value fails instead of
pausing. The operator (2026-11-22, the same feature as Zaptec PR 209) wants a configurable minimum charging current,
with every value below it pausing the charger.

## What Changes

- New public setting `min_charging_current` (int, default 6 A, the setter validates 1–32 A) over
  `cmd.config.set_int` / `cmd.config.get_int`, and in the packaged `config.json`.
- `initial_charging_current` gets the same routes and a setter (6–32 A, never below `min_charging_current`); raising
  the minimum above the effective initial current raises it in the same update, lowering it never changes it.
- An offered current below the minimum pauses the charger with the Easee pause-charging command, through the same
  paced command channel as `cmd.charge.stop`. The adapter remembers that pause; the next value at or above the
  minimum is written with the dedup bypassed, which resumes the charger. A `cmd.charge.stop`
  forgets the adapter's pause, so the adapter never adds a resume over a user stop.
- Values under 6 A are diverted in routing to the controller before cliffhanger's handler, and the message interface
  is renamed with a `.paused` suffix so that handler skips it; the adapter sends the current-session report itself.
  Values from 6 A up to the minimum reach the controller and pause there.
- Cliffhanger is not changed.

## Impact

- `src/internal/config`: the settings, defaults and ranges.
- `src/internal/easee`: the controller pause/resume, the thing factory's controller lookup.
- `src/internal/routing`: the config routes and the sub-6 A divert.
- `package/debian/.../defaults/config.json`: `min_charging_current: 6`.
