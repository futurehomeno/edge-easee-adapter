# A start at the initial current stays there on a hub without a load balancer

## What happened
Since issue 33 a `cmd.charge.start` with no known requested offered current writes
`initial_charging_current` (default 16 A) as the dynamic current. Nothing raises it afterwards
except `set_current`. On a hub without energy guard, a 32 A charger started this way charges at
16 A for the whole session; before, it started at its max current.

Found in the review of PR 236. The decision in issue 33 assumes a load balancer adjusts the current.

## Reproduce
No energy guard, charger max 32 A, no `set_current` since the adapter started or the last session
ended, `cmd.charge.stop` then `cmd.charge.start`: 16 A is sent and stays.

## What would close it
A decision: accepted for hubs without a load balancer (raise `initial_charging_current` there), or
the start goes to the max current when no balancer is present.
