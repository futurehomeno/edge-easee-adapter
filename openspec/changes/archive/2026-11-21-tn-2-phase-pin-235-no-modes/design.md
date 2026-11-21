# Design: no phase modes on TN 2-phase pins 2, 3, 5

## Decision
- `model.SettablePhaseModesFor(detected, gridType, phases)` is the one place that decides what a
  charger advertises: none for Easee grid type 30 (`WarningTN2PhasePin235`), the settable row of
  the FIMP pair otherwise. Thing creation and the grid-type observation both call it.
- Thing creation only has the FIMP pair in `State`, so `State` carries the raw type
  (`detectedGridType`, omitted when unknown), set from the charger config next to the pair. A state
  stored by an older adapter has no raw type; an unknown raw type is not 30, so it keeps the row of
  its FIMP pair, which is what that adapter advertised.
- The controller's phase-mode paths (`SetChargepointPhaseMode`, `legToRecord`,
  `ChargepointPhaseModeReport`) keep the FIMP-keyed `SettablePhaseModes`: cliffhanger's chargepoint
  service rejects `cmd.phase_mode.set` and sends no `evt.phase_mode.report` while `sup_phase_modes`
  is absent or empty (`phaseModeAwareController` reads the props on every call), so they are not
  reached on type 30.

## Why no row of its own
- Its legs are `NL1` and `NL3`; its two-phase mode would be `NL1L3`, which neither cliffhanger's
  phase modes nor Easee's `OutputPhaseType` (no `P2_T2_T3_T5`) can name.
- A row of `NL1`, `NL3` alone would let a balancer pick a leg but never the two-phase mode the
  charger drives in auto, and the report could not name the mode in effect. It is a warning state
  that is fixed on site; advertising nothing is what the adapter did before the TN 2-phase row and
  leaves the hub without a wrong promise.

## Why 36 and 38 keep the row
- 36 (`WARNING_TN_2_PHASE_PIN_2_3_4`) names the nominal pins 2, 3, 4: the row is its wiring.
- 38 (`WARNING_TN_2_PHASE_GND_FAULT`) is a fault on the ground conductor; nothing in Easee's
  enumeration says its phase pins move, so it keeps the nominal row.

## Republishing a change the FIMP pair does not show
The grid-type handler stopped when the FIMP pair matched the cache. 2 to 30 keeps TN with 2 phases
but changes the advertised modes, so the check also compares the settable modes of the observed raw
type with those of the raw type last advertised.

The raw type last advertised is a handler field (atomic, because it is written on the sender):
- seeded in `NewObservationsHandler` from the persisted state through `PhaseStore`, which already
  reads that state for the output phase. Thing creation built `sup_phase_modes` from that same
  state, so the first observation of an unchanged charger compares equal and a normal start sends
  no extra inclusion report;
- committed in the republish's `onPublished`, next to the cache's FIMP pair, so a dropped inclusion
  report is retried by the next observation, as the pair already is.

Rejected:
- Comparing with the props' `sup_phase_modes`: the props are replaced on the drain before the
  report lands, so a dropped republish of 2 to 30 would never be retried.
- Keeping the raw type in the cache next to the installation parameters: it changes
  `SetInstallationParameters` and every strict cache mock that sees a grid-type observation, for the
  same effect.
- Trusting the first observation as what creation advertised: a type-30 charger created from a
  state stored by an older adapter would keep the nominal row until a restart.

## Not covered
- With no `sup_phase_modes`, the handler's unforced phase-mode reports on a type-30 charger fail in
  cliffhanger (`aware phase modes is not supported`) and are logged as a warning by the report
  sender, as on every charger without the property before the TN 2-phase row.
- A charger created on type 30 that changes to 2 at runtime gains `sup_phase_modes` but not the
  phase-mode interfaces until the adapter restarts: cliffhanger derives them once, in `NewService`
  (the known limitation of `republishChargepointProps`).
