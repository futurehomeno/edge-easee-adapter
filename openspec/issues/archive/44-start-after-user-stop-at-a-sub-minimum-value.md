# 44: a start after a user stop resumes at a kept sub-minimum value

Found 2026-11-24 by the code review of PR 240 (`initial-current-session-start-only`).

Under a user stop, `cmd.current_session.set_current` only keeps the value for the next start, a sub-minimum one
included, and does not mark the adapter's own pause (A User Stop Holds Until A Start). Before PR 240 every start that
was not slow raised the kept value to `initial_charging_current`, so a kept 3 A started at 16 A. With PR 240 a start in
a session that already charged is a resume and keeps the value: it writes 3 A, which Easee treats as no charge. The
user's `cmd.charge.start` is answered as accepted, but the car does not charge.

This follows the spec as written ("the next `cmd.charge.start` starts at it"; "A start SHALL NOT check
`min_charging_current`"), but the outcome was not decided on its own.

Reproduce: charge at 10 A, `cmd.charge.stop`, `cmd.current_session.set_current 3`, `cmd.charge.start` in mode `normal`:
the dynamic current written is 3 A (`controller.go`, `StartChargepointCharging`).

Would close it: the operator deciding either

- a start writes the kept value as it is (current behaviour; the energy manager's budget is respected, the car waits), or
- a kept value below `min_charging_current` counts as the adapter's own pause, so the start writes
  `max(initial_charging_current, min_charging_current)` (one condition in `StartChargepointCharging` plus a scenario).

Decided 2026-11-24 (the operator): "when below minimum, chargins hould start with initial_charging". A kept value
below `min_charging_current` now starts like the adapter's own pause, at `max(initial_charging_current,
min_charging_current)`; slow sessions stay exempt, as there. Done in PR 240.
