# Skip the phase-mode report on a charger advertising no phase modes

## Why
- Since `tn-2-phase-pin-235-no-modes` an Easee on grid type 30 advertises no `sup_phase_modes`, and
  neither does a charger whose grid type is not yet detected.
- cliffhanger (v1.5.3) refuses `SendPhaseModeReport` on such a chargepoint with
  `<name>: aware phase modes is not supported`. The observation handler queues an unforced
  phase-mode report on every internal-mode and output-phase change, so the sender logs that error
  as a `Send report err` warning each time.

## What Changes
- The queued phase-mode report is skipped when the chargepoint service is not phase-mode aware
  (`IsPhaseModeAware()` false). The check runs when the report is sent, not when it is queued,
  because a topology republish queued in between can change the props.
- An output-phase change on such a charger still persists the reported leg.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/signalr/handler.go`
