# Session start or resume: open points of the initial-current change

Found 2026-11-24 while narrowing `initial_charging_current` to a session start
(`openspec/changes/initial-current-session-start-only`).

## 1. An adapter restart during a pause forgets that the session charged

Whether the session charged is kept in memory. After a restart the replayed charger state sets it again only when the
charger is charging. A charger paused at the restart (energy guard's stop, state `await_start`) reads as not charged.
The cached requested current is lost in the same restart, so a start goes to the charger maximum, as before. Only when
a `set_current` arrives before the start (kept under the user stop) is that start a session start and raised to 16 A,
the issue 42 behaviour, until the charger charges again.

Reproduce: charge, `cmd.charge.stop`, restart the adapter, `cmd.current_session.set_current 7`, `cmd.charge.start`:
the charger starts at 16 A.

Would close it: a decision that this is acceptable, or the session-charged value kept in the adapter's database like
the user stop.

Decided 2026-11-24 (the operator): accepted. An adapter restart is very rare; the value stays in memory.

## 2. A `set_current` that starts the charger makes the next start a resume

Easee resumes a waiting charger on any dynamic-current write. A `set_current` of 8 A made before the first
`cmd.charge.start`, outside a user stop, starts the charger; once the charging observation arrives, the next start is a
resume at 8 A, not a session start at 16 A. The rule says the `set_current` does not make it a resume, and defines a
session start as "the charger has not charged". The change follows the second: the charging decides.

Would close it: the operator confirming that a charger which already charged at 8 A resumes at 8 A.

Decided 2026-11-24 (the operator): right, the charging decides. A charger that already charged at 8 A resumes at 8 A.

## 3. The adapter's own pause outlives the session

The adapter's pause mark ends only on the next value at or above `min_charging_current` or on `cmd.charge.stop`, not
when the car is disconnected. A start in the next session, with nothing cached, then starts at the initial charging
current (16 A) instead of the charger maximum.

Reproduce: `set_current 3` (pause), unplug, plug in a car, `cmd.charge.start`: 16 A, not the charger maximum.

Would close it: a decision whether the session end also ends the adapter's pause (Zaptec drops it on a disconnect).

Decided 2026-11-24 (the operator): "Unplugging is session finished so session specific values should be reset to
default." Fixed in PR 240: the session-finished observation ends the adapter's pause, and an unplugged car also drops a
write or phase bounce still waiting in the command channel (requirement "A Session End Resets The Session Values").

## Closed

All three points decided 2026-11-24 and done in PR 240.
