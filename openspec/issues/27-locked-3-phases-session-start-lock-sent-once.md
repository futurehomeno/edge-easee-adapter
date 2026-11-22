# use_locked_3_phases: the lock at the session start is sent once

Found in the PR 230 review (change `add-use-locked-3-phases`).

The handler asks for the three-phase lock when a session starts on auto, and not again on a change inside
that session (a setting changed during a session must force nothing). So the session runs on auto, as it
would without the setting, when that one request is lost:

- Easee answers the phase mode write with an error, or the report queue is full (both are logged).
- The session starts while a topology update waits for its inclusion report: the handler still reads the old
  advertised grid, finds fewer than two settable modes and skips the lock. Seen as possible at start, when
  the charger config read failed and the stored topology is old or empty.

A later single-to-three-phase request from the hub locks it. The next session is locked at its start.

Observe: setting on, make the phase mode write fail at plug-in (Easee API unreachable); the log shows
`Three-phase lock: wanted=true on auto` once and `Send report err`, and the charger config keeps phase mode 2
for the whole session.

Closes when: the user accepts this, or the lock is asked again once when the write failed or the topology
update committed.
