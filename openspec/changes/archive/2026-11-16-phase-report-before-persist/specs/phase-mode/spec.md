## MODIFIED Requirements

### Requirement: Advertised Modes Do Not Follow The Observed Leg
`sup_phase_modes` SHALL stay the settable set for the charger's grid type and phase count when the
charger reports a single leg, because an Easee picks its leg per session and a list narrowed to the
last leg makes cliffhanger reject a request for another one. The reported leg SHALL be persisted for
the phase-mode report without republishing the inclusion report. The write SHALL go with the queued
phase-mode report, after it is published and not on the observation drain, so a write that blocks or
fails SHALL NOT hold the report back.

#### Scenario: an IT charger changes leg
- **GIVEN** a charger on an IT grid with 3 phases advertising `L1L2`, `L2L3`, `L3L1` and `L1L2L3`, last seen on `L1L2`
- **WHEN** it reports output phase `P1_T2_T4_IT` (`L3L1`)
- **THEN** all four modes stay advertised, `L3L1` is persisted and no inclusion report is sent

#### Scenario: the leg cannot be saved
- **WHEN** a charger reports a new leg and writing it to the thing state blocks or fails
- **THEN** the observation is handled without waiting for the write and the phase-mode report is still published

#### Scenario: the write is still blocked
- **WHEN** a charger reports a new leg and the write to the thing state has not returned
- **THEN** the phase-mode report is already published
