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
restart, so such a start goes to the charger maximum unless a `set_current` arrives first (issue 43).

## The adapter's own pause

The start after the adapter's own pause starts at `max(initial_charging_current, min_charging_current)`, capped by the
charger maximum, as Zaptec does. The start only reads the pause mark; the mark ends as before, on the next value at or
above the minimum or on a stop.
