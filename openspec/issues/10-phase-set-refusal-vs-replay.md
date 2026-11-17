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
- PR #219 review, finding 5: acting on the charger's `CommandResponse` instead of the timer.
  Investigated 2026-11-17, see below: the answer cannot tell an applied mode from an ignored one.

## Field data (2026-11-17)

Debug logs of hubs `7a18325dc9eaa3b7` (EH496315) and `9c1edbb3d437c8f1` (EH152843), 5832 unique
`CommandResponse` answers:

- Payload: `serialNumber`, `id`, `ticks`, `timestamp`, `deliveredAt`, `wasAccepted`, `resultCode`,
  `comment`. Easee documents it only for AMQP (https://developer.easee.com/docs/amqp-commands). `id` 38 is
  the phase mode, 48 the dynamic current, 47 the max current, 30 the cable lock.
- All 5832 answers say `wasAccepted: true`, `resultCode: null`. No refused answer was ever logged,
  so its format is unknown. The 53 phase-mode answers arrived 0-2 s after the send.
- `wasAccepted: true` does not mean the charger applied the value: in the 2026-09-20 customer log
  17 of 17 resumes were ignored by the charger while each was answered as accepted
  (`pace-offered-current-writes`). The answer is a delivery receipt.
- The internal-mode echo (observation 38) arrived 1-2 s after the send in 25 of 25 sends where the
  log could show it. No send without an echo was seen, so "accepted but never applied" has no field
  evidence for the phase mode yet.

Result: a handler for a refused answer would cover a case never seen and miss the one this issue is
about. The echo is the signal that tells the two apart: a re-sync could be skipped when the echo
arrived within the 15 s, which would leave the re-subscribe to the sends the charger did not echo.

Open question: skip the re-sync when the echo has arrived?
