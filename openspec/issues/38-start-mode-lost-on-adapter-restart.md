# The remembered start mode is lost on an adapter restart

## What happened
Change `slow-session-resume` remembers the mode of a session's last start that named one, so a
start without a mode (energy guard resuming a session it paused) stays slow. The mode lives in the
per-charger cache beside the requested offered current. The cache is in memory only
(`cache.NewCache`), and the requested offered current is empty after a restart too.

## How to reproduce
Start in mode `slow`, restart the adapter, let energy guard stop and start the charger. The start
has no mode and no remembered mode, so it is raised to `initial_charging_current` (16 A).

## Doubt
Is a restart in the middle of a slow session rare enough to accept? Persisting the mode would need
its own storage and a rule for when a stored mode is stale (the session-finished observation can be
missed while the adapter is down).

## What would close it
- The operator accepts it: archive this issue. Or
- persist the mode per charger with the session, cleared by the session-finished observation or a
  new session id after the restart.
