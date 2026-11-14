# TN 2-phase pin variants share one phase mode row

The phase mode matrix is keyed by FIMP grid type and phase count, so all four Easee TN 2-phase grid
types map to the one TN 2-phase row (`NL1`, `NL2`, `NL1L2`), built for the nominal pins 2, 3, 4.

Doubts:
- `WarningTN2PhasePin235` (30) wires pins 2, 3, 5, i.e. `NL1` and `NL3`. The row advertises `NL2`
  and `NL1L2`, which that wiring cannot deliver. There is no FIMP `NL1L3` (cliffhanger) nor an Easee
  `P2_T2_T3_T5` output phase, so its two-phase mode cannot be named at all.
- `WARNING_TN_2_PHASE_GND_FAULT` (38) does not say which pins carry the phases.
- Both are warning states that should be fixed on site; whether they deserve their own rows (which
  means passing the Easee grid type, not only the FIMP one, to the mapper) is open.
- The pin-to-leg reading (pin 2 N, pin 3 L1, pin 4 L2) comes from the adapter's own
  `outputPhaseModes`; Easee's developer docs name terminals but do not map them to legs.
