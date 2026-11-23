# 40: races around a user stop left open after PR 235

Found while fixing the PR 235 review findings on the user stop (`configurable-minimum-current`).

- A value under 6 A whose write is dropped by the re-check in `dispatchPair` (a user stop landed after the caller's
  check) is not kept as the requested current, so the next `cmd.charge.start` starts at the value before it.
- `StartChargepointCharging` reads the stop, clears it and puts it back when the write fails, in three storage calls.
  An observation that ends the stop between the clear and the put-back is undone by the put-back. Both need a start
  that Easee refuses, while the car is unplugged or charging starts elsewhere.
- A deferred start whose write fails later leaves the stop ended (now in the spec): the start was answered as
  accepted, so the next value resumes the charger.

Reproduce: only by interleaving the calls in a test; no hub has shown it.

Closes when a hub log shows one of them, or the user stop moves behind one lock shared by the controller and the
handler.
