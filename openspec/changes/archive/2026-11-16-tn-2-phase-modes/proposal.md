# A TN 2-phase charger reports its phase modes

## Why
- Easee reports four TN 2-phase grid types (2, 30, 36, 38); `ToFimpGridType` maps all of them to
  TN with 2 phases, but the phase mode matrix has rows only for 1 and 3 phases.
- Every lookup logs `phase modes mapper: unsupported number of phases: 2` and returns no modes, so
  the inclusion report omits `sup_phase_modes` and every `cmd.phase_mode.get_report` fails with
  `chargepoint: aware phase modes is not supported`, although the charger reports `PhaseMode=NL2`.

## What Changes
- The matrix gains a TN 2-phase row for the nominal wiring (neutral on pin 2, L1 on pin 3, L2 on
  pin 4): single-phase lock `NL1`, `NL2`; auto `NL1`, `NL2`, `NL1L2`; three-phase lock `NL1L2`.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/model/mapper.go`
