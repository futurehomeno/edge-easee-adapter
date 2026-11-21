# Design: TN 2-phase legs

## Decision
One TN 2-phase row, for the wiring Easee names as its nominal 2-phase grid,
`TN2PhasePin234` (value 2):

| Easee internal mode | FIMP modes |
| --- | --- |
| 1 single-phase lock | `NL1`, `NL2` |
| 2 auto | `NL1`, `NL2`, `NL1L2` |
| 3 three-phase lock | `NL1L2` |

## Source
- Easee developer documentation, https://developer.easee.com/docs/enumerations:
  `detectedPowerGridType` 2 `TN2PhasePin234`, 30 `WarningTN2PhasePin235`,
  36 `WARNING_TN_2_PHASE_PIN_2_3_4`, 38 `WARNING_TN_2_PHASE_GND_FAULT`; `OutputPhaseType`
  names the terminals the charger drives.
- The adapter's own terminal-to-leg mapping (`outputPhaseModes`): `P1_T2_T3_TN` is `NL1`,
  `P1_T2_T4_TN` is `NL2`, `P2_T2_T3_T4_TN` is `NL1L2`. So pin 2 is neutral, pin 3 L1, pin 4 L2;
  a grid on pins 2, 3, 4 offers exactly `NL1`, `NL2` and `NL1L2`.
- A TN 2-phase charger in the field reports `PhaseMode=NL2` (`P1_T2_T4_TN`), consistent with pins
  2, 3, 4.
- The row follows the 3-phase row's shape: single lock lists every single leg, auto adds the
  multi-phase mode, the three-phase lock uses all wired legs. cliffhanger accepts `NL1`, `NL2`,
  `NL1L2`.

## Not covered
- The matrix is keyed by FIMP grid type and phase count, so the four Easee 2-phase values share the
  row. `WarningTN2PhasePin235` (pins 2, 3, 5: `NL1`, `NL3`) has no two-phase mode in either Easee's
  `OutputPhaseType` or cliffhanger (`NL1L3` does not exist), and the ground-fault variant does not
  name its pins. Splitting by Easee grid type would thread that value through every caller for
  warning states; recorded as an issue instead.
