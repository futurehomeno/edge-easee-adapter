# A replay before the echo undoes a phase set

Found in the review of PR #214 (change `phase-set-before-echo`). Read from the code, not seen in a
log.

The sent target is cached with the replaced observation's timestamp, and the cache accepts an equal
timestamp. A replay that lands before the echo (a SignalR reconnect, a `Resync`, or the 5-minute
stale watchdog) carries the pre-send mode at that same timestamp. It replaces the target, so a
request for the old mode in that gap is dropped as a no-op again.

Closed by `phase-set-resync` (3.3.1): a mode Easee accepted (200/202) and the charger never applied
left the target cached, so a repeated request was not sent until some replay arrived. A mode sent
without a session restart now arms the 15 s re-sync, whose replay restores the real mode. A restart
is re-synced by its own stop and resume (`phase-resync-review`).

Left open:
- That re-sync is itself a replay: an echo slower than 15 s opens the gap above until it lands. A
  phase mode sent while a re-sync is pending keeps that one, so its replay can come sooner than
  15 s after the send.
- After the replay restored the real mode, the report still names the recorded request until a newer
  output-phase or phase-mode observation arrives, as it did up to 3.2.5. Energy guard asks again
  only once it sees another leg. PR #219 review, finding 2: a replay that replaces a sent target at
  an equal timestamp could also drop the request record. The report would then follow the charger,
  but a charger that keeps refusing would get the set repeated on every energy guard tick, each
  with a re-subscribe, and a replay that only beat a slow echo would flip the report twice.
- A charging charger whose pause fails, or that has no current to resume at, arms no re-sync, as on
  3.3.0.
- How long is the gap between a send and its echo, and how often does a charger refuse
  `set_phase_mode` after Easee accepted it? `receiver.CommandResponse` only logs the charger's
  answer at debug level. PR #219 review, finding 5: acting on a refused answer would replace the
  timer and the re-subscribe after every accepted set. It needs the payload of a refused answer
  and a way to match it to the charger and the command.
