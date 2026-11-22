Resolved: 2026-11-22, the user: accepted. A mode 3 chosen in the Easee app can be overwritten once in this case; the lock memory is not changed.

# use_locked_3_phases: a mode 3 observed after the release keeps the lock memory

Found in the PR 230 review, round 4 (change `add-use-locked-3-phases`).

The handler remembers that a mode 3 on the charger is the adapter's lock. The memory ends when the release
is queued at the session end, and a mode 3 observed while the setting is on sets it again. So a mode 3
observation that comes after the session end but before the charger shows auto (a re-sync replay while Easee
has not applied the release yet) sets the memory, and the later observation of auto does not end it.

Harm needs two more steps by the user: the setting is turned off, and mode 3 is then chosen in the Easee
app. The next session start writes auto over that choice, once. After that the memory is off and a mode 3
set in the app stays.

Kept on purpose: the same observation is what asks again for a release that Easee accepted but the charger
never applied. Not remembering a mode 3 seen outside a session (the reviewer's fix) would leave that lock on
the idle charger for good once the setting is off.

Observe: setting on, unplug on mode 3. The log shows `Three-phase lock: wanted=false`, then `Phase mode=`
with `(3)` and later `(2)`. Turn the setting off, set mode 3 in the Easee app, plug in:
`Three-phase lock: wanted=false` is logged again.

Closes when: the user accepts this, or an observed mode other than 3 on a charger outside a session also
ends the memory.
