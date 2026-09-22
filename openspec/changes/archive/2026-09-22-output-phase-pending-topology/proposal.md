# Build the output-phase list from the pending topology

## Why
Since #176 a grid-type observation writes the cache only after its inclusion report lands. The
output-phase handler still read the grid type and phase count from that cache, so an output phase
arriving while the topology republish was queued saw the old topology:
- a charger created without a stored grid type (grid type and output phase in one initial batch)
  compared two empty lists, stored the phase without republishing, and the hub kept advertising the
  default single-phase leg until the next restart;
- on a real topology change it rebuilt `sup_phase_modes` for the old topology and overwrote the new
  list the grid-type republish had just set.

## What Changes
- The output-phase handler reads grid type and phase count from the chargepoint props, which a
  topology republish replaces synchronously, instead of the cache.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/signalr/handler.go`
