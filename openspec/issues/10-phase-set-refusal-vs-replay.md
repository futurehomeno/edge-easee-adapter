# A replay before the echo undoes a phase set, and only a replay undoes a refused one

Found in the review of PR #214 (change `phase-set-before-echo`). Read from the code, not seen in a
log.

The sent target is cached with the replaced observation's timestamp, and the cache accepts an equal
timestamp. That has two effects, and they pull opposite ways:

- A replay that lands before the echo (a SignalR reconnect, the `Resync` 15 s after a session
  bounce, or the 5-minute stale watchdog) carries the pre-send mode at that same timestamp. It
  replaces the target, so a request for the old mode in that gap is dropped as a no-op again.
- Easee may accept a command (200/202) that the charger then refuses. The value does not change, so
  no observation follows. `receiver.CommandResponse` only logs the charger's answer at debug level,
  so nothing but such a replay restores the real mode. Until a replay arrives, a repeated request
  for the refused mode is not sent. Before this change it was sent again.

Stamping the target past the replaced timestamp would fix the first effect. It would also make the
second permanent until the charger reports a different mode.

Open questions: how long is the gap between a send and its echo, and how often does a charger
refuse `set_phase_mode` after Easee accepted it? Should a refused phase-mode `CommandResponse` drop
the sent target, or trigger a `Resync`?
