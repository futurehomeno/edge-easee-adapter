# The three-phase lock after an adapter restart depends on the charger config read at start

`use_locked_3_phases` (change `add-use-locked-3-phases`) locks or releases three phases when a charger state
observation changes the cached state, from the cached internal phase mode. After an adapter restart that
mode is not empty: thing creation seeds it from the charger config read over REST
(`src/internal/easee/thing.go`, `thingCache.SetPhaseMode`) before the SignalR handler is registered, so the
replayed state counts as a change against the real mode whatever order the observations come in. (The first
text of this issue said the cache was empty; it is not.)

Left: when that read fails, the mode from the stored thing state is used and can be old.

- Stored auto, charger in a session on mode 3: mode 3 is written again, with one needless stop/resume if it
  is charging.
- Stored auto, charger idle on mode 3 (the session ended while the adapter was down): nothing is sent; with
  the setting on the next session is locked at plug-in anyway and released at its end.
- Charger in a session on auto with the setting on (turned on during the session, or the lock at the start
  failed, issue 28): the replayed state counts as a session start, so mode 3 is written with one stop/resume
  if it is charging, although a setting changed during a session should force nothing. Not told apart on
  purpose: the same first-state rule locks a session that started while the adapter was down and releases
  a mode 3 left on an idle charger.

Not triggered from the phase mode observation on purpose: Easee can accept a mode the charger never applies,
the 15 s re-sync then replays the old mode, and a trigger there would re-send mode 3 every 15 s.

Observe: restart the adapter with the setting on, the charger idle on mode 3 and the Easee API unreachable
at start; read the log for `Three-phase lock:`.

Closes when the user accepts this, or the lock is also checked once at the first phase mode observation that
differs from the seeded mode.
