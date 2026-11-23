Resolved: 2026-11-22, the user: accepted. A restart between turning the setting off and the end of a locked
session leaves mode 3 on the idle charger until the next single-to-three-phase request; the lock ownership is
not stored.

# use_locked_3_phases: who placed the lock is forgotten by an adapter restart

Found in the PR 222 review (change `add-use-locked-3-phases`).

The handler remembers that it placed the three-phase lock (`lockWasOn`) only in memory. When the setting is
turned off during a locked session and the adapter restarts before that session ends, the new handler starts
with no memory of the lock; the replayed mode 3 does not set it either, because the setting is off. At the
disconnect neither the setting nor the memory asks for the release, so mode 3 stays on the idle charger and
the next car starts locked on three phases. Stated in the change's design as a limitation of the in-memory
memory.

Observe: setting on, start a session (mode 3 is sent), turn the setting off, restart the adapter, unplug the
car; the log shows no `Three-phase lock: wanted=false` and the charger config keeps phase mode 3.

Closes when: the user accepts this, or the ownership is stored in the thing state and read at the handler's
start.
