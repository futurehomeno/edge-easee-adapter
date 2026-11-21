# A TN 2-phase charger on pins 2, 3, 5 advertises no phase modes

## Why
- Easee grid type 30 (`WarningTN2PhasePin235`) wires neutral on pin 2, L1 on pin 3 and L3 on pin 5.
  It maps to FIMP TN with 2 phases, so since `tn-2-phase-modes` it advertises the nominal TN 2-phase
  row `NL1`, `NL2`, `NL1L2`. It can deliver neither `NL2` nor `NL1L2`, and FIMP has no `NL1L3` to
  name its two-phase mode.
- A charger on type 30 accepts a set to `NL2` while it draws on L3 (issue 2).

## What Changes
- Grid type 30 advertises no settable phase modes (no `sup_phase_modes`), as it did before the TN
  2-phase row existed. Its FIMP grid type and phase count stay TN with 2 phases; the
  grid-type-fault alarm already raised for it stays the report.
- Grid types 2, 36 and 38 keep the nominal TN 2-phase row.
- The persisted charger state carries Easee's raw grid type next to the FIMP pair, so thing
  creation can tell the variants apart. A state stored by an older adapter falls back to the row of
  its FIMP pair.
- The grid-type observation republishes when the settable modes of the raw type change, also when
  the FIMP pair does not (2 to 30 and back), but not on a normal start of an unchanged charger.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/model/mapper.go`, `src/internal/easee/thing.go`,
  `src/internal/easee/controller.go`, `src/internal/easee/connector.go`,
  `src/internal/signalr/handler.go`
