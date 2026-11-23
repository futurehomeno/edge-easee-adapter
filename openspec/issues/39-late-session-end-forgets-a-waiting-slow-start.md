# A late end of the previous session forgets a slow start still waiting to be sent

## What happened
Found in the review of PR 237 (indivisus, 2026-11-23, "A late prior-session clear erases a queued
slow start"). The slow-start memory (`slowSince` in `controller.go`) is the time the start was
requested. A slow start sent within `offered_current_wait_time` of the previous write waits in the
channel. If the session-finished observation of the previous session is handled in that wait, it
clears the cached current with a time later than `slowSince`, and a `set_current` before the send
forgets the memory, although it displaces and inherits the waiting start.

## How to reproduce
1. Send any write, then within the wait time `cmd.charge.start` with mode `slow`: it waits.
2. Before it is sent, a state observation that ends the session (Completed, Disconnected, …)
   arrives, then a `set_current`.
3. Later `cmd.charge.stop`, then `cmd.charge.start` with no mode: offered
   `initial_charging_current` (16 A, capped by the max current) instead of the session's current.

## Why it is not fixed in PR 237
Four events in that order inside one wait window; the cost is one resume at the 16 A floor until
energy guard or the balancer sets the current again. Stamping the memory with the time the start goes
out (the bot's fix) trades it for the opposite race: a session that really ends while the start waits
keeps the memory. Neither closes it without knowing which session the charger is in.

## What would close it
The memory is tied to a session id or start observed from the charger rather than to the cached
current, or the operator accepts the race: archive this issue.
