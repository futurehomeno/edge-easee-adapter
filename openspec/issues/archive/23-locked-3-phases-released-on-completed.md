Resolved: 2026-11-22, the user: `finished` (Easee "Completed") counts as a session in progress, so the lock is kept and nothing is written there; `disconnected`, `error` and `requesting` end it. Change `add-use-locked-3-phases`, PR 230.

# use_locked_3_phases: `completed` releases the lock

Found while adding `use_locked_3_phases` (change `add-use-locked-3-phases`).

The session rule counts `completed` (FIMP `finished`) as the end of a session, like
`disconnected`. Easee also reports `completed` when the car pauses or is full while still plugged in
(`docs/easee-enumerations.md`: "Car has paused/stopped charging"). The adapter then writes auto. A
car that resumes from there goes to `charging` on auto, the start rule writes mode 3 again, and the
controller bounces the session (stop, mode write, resume, about 30 s) to apply it.

How to observe: setting on, a three-phase car charging on mode 3; let it finish or pause from the car
and resume. The log shows `Three-phase lock: wanted=false on three phase` at
`Completed`, then `wanted=true on auto` and a pause/resume when it charges again.

Alternative: only `disconnected` ends the lock (keep it through `completed`, `error`,
`awaiting_authentication`).

Closes when: the user decides which states end the lock.
