# use_locked_3_phases: an error without a car locks an idle charger

Found while making only `disconnected` release the lock (the user, 2026-11-22; change
`add-use-locked-3-phases`).

`error` and `requesting` now count as a session in progress, because the car is still plugged in. If Easee
also reports `error` (`ChargerStateError`) with no car plugged in, a change from `disconnected` to `error`
on auto counts as a session start: with the setting on, mode 3 is written to an idle charger and stays
until the next `disconnected`. A car plugged in meanwhile starts on the lock, which a car refusing three
phases may not charge on.

Unverified whether Easee reports `error` without a car in practice.

Observe: setting on, charger idle on auto. A debug log with `State=` going from disconnected to error
followed by `Three-phase lock: wanted=true` shows it.

Closes when field logs show whether `error` occurs without a car, or the user accepts it.
