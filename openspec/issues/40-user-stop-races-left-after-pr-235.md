# 40: user-stop races and edges left open after PR 235

Found while fixing the PR 235 review findings on the user stop (`configurable-minimum-current`).

- A value under 6 A whose write is dropped by the re-check in `dispatchPair` (a user stop landed after the caller's
  check) is not kept as the requested current, so the next `cmd.charge.start` starts at the value before it.
- `StartChargepointCharging` reads the stop, clears it and puts it back when the write fails, in three storage calls.
  An observation that ends the stop between the clear and the put-back is undone by the put-back. Both need a start
  that Easee refuses, while the car is unplugged or charging starts elsewhere.
- A deferred start whose write fails later leaves the stop ended (now in the spec): the start was answered as
  accepted, so the next value resumes the charger.
- A value kept under a user stop is dropped by a session-finished state (completed, error, offline, unknown,
  authentication), like every requested current since #31, so the next start begins at the user's max or the initial
  current (the rule for a start with no known current, the user, 2026-11-23). Not changed: the session it was meant
  for has ended.
- Not a user-stop race, but left from the same review: two values under the minimum sent at the same moment both
  take the pause mark; when the first one's stop then fails, it puts back "no pause" although the second one's stop
  went out. The next valid value is then not forced, so the dedup can skip it when it equals the value before the
  pause, and the charger stays paused until a different value or the wait time passes. `CompareAndSwap` does not close
  it (the mark is still set when the first one rolls back); it needs the mark and the send under one lock.

Reproduce: only by interleaving the calls in a test; no hub has shown it.

Closes when a hub log shows one of them, or the user stop moves behind one lock shared by the controller and the
handler.
