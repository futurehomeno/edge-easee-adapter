# A replay before the echo undoes a phase set

Found in the review of PR #214 (change `phase-set-before-echo`). Read from the code, not seen in a
log.

The sent target is cached with the replaced observation's timestamp, and the cache accepts an equal
timestamp. A replay that lands before the echo (a SignalR reconnect, a `Resync`, or the 5-minute
stale watchdog) carries the pre-send mode at that same timestamp. It replaces the target, so a
request for the old mode in that gap is dropped as a no-op again.

Closed by `phase-set-resync` (3.3.1): a mode Easee accepted (200/202) and the charger never applied
left the target cached, so a repeated request was not sent until some replay arrived. A sent mode
now arms the 15 s re-sync, whose replay restores the real mode.

Left open:
- That re-sync is itself a replay: an echo slower than 15 s opens the gap above until it lands.
- After the replay restored the real mode, the report still names the recorded request until a newer
  output-phase or phase-mode observation arrives, as it did up to 3.2.5. Energy guard asks again
  only once it sees another leg.
- How long is the gap between a send and its echo, and how often does a charger refuse
  `set_phase_mode` after Easee accepted it? `receiver.CommandResponse` only logs the charger's
  answer at debug level.
