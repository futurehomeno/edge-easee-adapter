# Design

## The session-start signal

A session start is a `cmd.charge.start` before the charger has charged in the session. The adapter already has three
candidate signals; only the charger state tells whether the charger charged:

- The first `cmd.charge.start` since the session-finished observation (the `slowSince` pattern) misses the common
  case. An Easee charger starts charging on its own when the car is connected, and energy guard sends no
  `cmd.charge.start` until it resumes a charger it paused. In the hubsim it3-easee log of issue 42 the session starts
  at the plug-in at 17:05:05, energy guard stops it at 17:15:15, and its first `cmd.charge.start` at 17:25:28 is
  already a resume.
- Easee's session observations (223 start, 129 stop, kept in the session storage) span plug-in to unplug and say
  nothing about charging.
- The user-stop mark and the adapter's pause mark miss a start while charging, a pause made in the Easee app or by
  the car, and they would make a start after a stop before any charging a resume.

So the cache keeps `sessionCharged`, a timestamped bool like every cached value. `handleChargerState` sets it on a
charging observation and clears it where it clears the cached requested current: the session-finished observation
(disconnected, completed, error, offline, unknown, awaiting authentication, de-authenticating). The value is in
memory: after an adapter restart the replayed charger state sets it again when the charger is charging. A charger
paused at the restart reads as not charged until it charges again; the cached requested current is lost in the same
restart, so such a start goes to the charger maximum unless a `set_current` arrives first. The operator accepted this
(issue 43, archived).

## The adapter's own pause

The start after the adapter's own pause starts at `max(initial_charging_current, min_charging_current)`, capped by the
charger maximum, as Zaptec does. The start only reads the pause mark; the mark ends as before, on the next value at or
above the minimum or on a stop.

## A session end resets the session values

The operator, 2026-11-24: "Unplugging is session finished so session specific values should be reset to default". The
handler's session-finished branch now calls `EndSession(unplugged)` on the controller (wired by the connector) before it
clears the cached requested current and `sessionCharged`; a write is stamped with the time it left, so one being sent
cannot refill the cleared current. Audit of what the adapter keeps per session:

| Value | Reset |
|---|---|
| Cached requested offered current | already, at every session end |
| `sessionCharged` | this change, at every session end |
| The adapter's own pause (`offeredPause`) | now, at every session end. The requested current is cleared there too, so the dedup cannot swallow the next value |
| Slow-session memory (`slowSince`) | now, at every session end, even when a waiting write refills the cached current afterwards (issue 39) |
| User stop (database) | already, when the car is disconnected; completed, error and offline keep it (issue 33) |
| `use_locked_3_phases` lock | already, when the car is disconnected (PR 233) |
| A write or phase bounce waiting in the command channel | now, dropped when the car is disconnected. Sent later, it would write the finished session's current back into the cache |

A waiting command is dropped only on a disconnect. On an offline, completed or error report the car is still there,
and the command is the hub's last word: dropping a cut there could leave the charger above the balancer's budget, and
dropping a bounce's resume could leave it paused. The pace window (`sentAt`), the phase mode sent last and the clamp
warning belong to the charger, not to the session, and stay as they are.

