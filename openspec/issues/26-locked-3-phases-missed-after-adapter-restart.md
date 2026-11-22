# The three-phase lock is moved only on a state change seen with the internal mode cached

`use_locked_3_phases` (change `add-use-locked-3-phases`) locks or releases three phases when a charger state
observation changes the cached state, and only if the internal phase mode is already in the cache. After an adapter
restart the cache is empty, so the outcome depends on the order SignalR replays the observations in (unknown):

- State replayed before phase mode 38: nothing is sent. A session that ended while the adapter was down leaves the
  charger on mode 3 until its next session ends; a session running on auto stays on auto.
- Phase mode first: the replayed state counts as a change and the lock is moved as specified.

Not triggered from the phase mode observation on purpose: Easee can accept a mode the charger never applies, the
15 s re-sync then replays the old mode, and a trigger there would re-send mode 3 every 15 s.

Observe: restart the adapter with the setting on while the charger is idle on mode 3, and read the log for
`Three-phase lock:`.

Closes when the replay order is known and, if the state comes first, the lock is also checked once after the first
phase mode observation of a connection.
