# 34: the controller lookup is filled inside `ThingFactory.Create`

Found in the review of PR 235 (`configurable-minimum-current`).

`ThingFactory.Create` stores the new controller in `ThingFactory.controllers`, which the routing divert of offered
currents under 6 A reads. cliffhanger 1.5.3 documents that `Create` must leave nothing behind: `RebuildChangedThings`
(`adapter/drift.go`) builds a prospective thing and throws it away when the service topology is unchanged. The thrown
away controller would then stay in the lookup, and a value under 6 A would be written through it: outside the live
controller's 20 s pacing, and with the pause remembered where the next valid current does not look.

Not reachable today: the adapter calls `SyncThings` only, never `RebuildChangedThings`.

Closes when the adapter starts calling `RebuildChangedThings`: the lookup must then be filled from the connector's
`Connect` (and emptied in `Disconnect`), or cliffhanger accepts values under 6 A and the divert goes away.
