# Start without a known offered current goes to the max

## What happened
`StartChargepointCharging` (`src/internal/easee/controller.go:787-797` on release_3.4) treats a
`RequestedOfferedCurrent` ≤ 0 as unknown and starts at the charger's max current, or
`maxCurrentValue` (32 A) when that is unknown too. After an adapter restart, or a session the
charger finished, a Start therefore offers up to 32 A before any load balancer has set a budget.

Found while checking cliffhanger PR #289 (session current passes 0–5 A). The 0–5 A pause itself
is PR 235 (`min_charging_current`, pauses without writing 0 into `RequestedOfferedCurrent`).

## Decision (the user, 2026-11-23)
With no known current, Start begins at 16 A at most, then the load balancer adjusts it:
`min(InitialChargingCurrent(), charger max)`; `initial_charging_current` defaults to 16
(`src/internal/config/config.go:16`).

## Reproduce
Fresh adapter, charger max 32 A, no `set_current` yet, `cmd.charge.start`: 32 A is sent.

## What would close it
The fallback above, with a test: Start with no requested current and max 32 A sends 16 A; max 10 A
sends 10 A.
