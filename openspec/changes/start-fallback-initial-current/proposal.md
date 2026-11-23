# A start with no known current begins at the initial charging current

## Why
- `cmd.charge.start` with no cached requested offered current (after an adapter restart, or a
  session the charger finished) starts at the charger's max current, or 32 A when that is unknown
  too. It offers up to 32 A before any load balancer has set a budget (issue 33).

## What Changes
- With no known requested offered current, the start current is `initial_charging_current`
  (default 16 A), capped at the cached max current, or at 32 A when that is unknown. The load
  balancer adjusts it afterwards (the user, 2026-11-23).
- The other start rules stay: a known current, the normal-mode floor, slow mode.

## Impact
- Affected specs: `charging-control`
- Affected code: `src/internal/easee/controller.go`
