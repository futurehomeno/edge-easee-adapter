## Why

On beta hub 8ee88ca8 (IT grid, two Easee Home, upgraded 2.8.0 → 3.2.5 on 2026-10-23) energy guard's
`cmd.phase_mode.set` was rejected twice on 2026-10-25:

```
E failed to set chargepoint phase mode: chargepoint: unsupported phase mode: L1L2  (ad:2, 17:34:10)
E failed to set chargepoint phase mode: chargepoint: unsupported phase mode: L3L1  (ad:1, 17:38:55)
```

Since #134 `sup_phase_modes` offers one single-phase leg, the last one observed, on the premise that
an Easee always uses the leg it is wired to. These chargers do not: under 2.8.0, with no phase command
sent at all, both alternated between T2T3 (`L1L2`) and T2T4 (`L3L1`) session by session. Every switch
re-advertised the list, and cliffhanger rejected the leg energy guard had requested (and seen
accepted) earlier, although it maps to the same Easee setting (single phase). The setter
already answers a request for a leg the charger is not on (#159), and energy guard learns the leg
from that answer, so the narrowing only adds the rejection.

## What Changes

- `sup_phase_modes` is the settable set for the grid type and phase count (the existing
  Settable Phase Modes requirement), whatever leg the charger last used.
- A new output phase is persisted for the report but no longer republishes the inclusion report.

## Capabilities

### Modified Capabilities
- `phase-mode`: the advertised modes do not follow the observed leg.

## Impact

- `internal/model/mapper.go` (`AdvertisedPhaseModes` removed), `internal/signalr/handler.go`,
  `internal/easee/thing.go`, `internal/easee/controller.go`. No config change.
