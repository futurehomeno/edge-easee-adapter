## ADDED Requirements

### Requirement: Advertised Modes Do Not Follow The Observed Leg
`sup_phase_modes` SHALL stay the settable set for the charger's grid type and phase count when the
charger reports a single leg, because an Easee picks its leg per session and a list narrowed to the
last leg makes cliffhanger reject a request for another one. The reported leg SHALL be persisted for
the phase-mode report without republishing the inclusion report.

#### Scenario: an IT charger changes leg
- **GIVEN** a charger on an IT grid with 3 phases advertising `L1L2`, `L2L3`, `L3L1` and `L1L2L3`, last seen on `L1L2`
- **WHEN** it reports output phase `P1_T2_T4_IT` (`L3L1`)
- **THEN** all four modes stay advertised, `L3L1` is persisted and no inclusion report is sent

## MODIFIED Requirements

### Requirement: Output Phase Uses The Advertised Topology
The output-phase handler SHALL map the reported leg with the grid type in the chargepoint
specification, not the cache, because a topology republish replaces the specification at once but
commits the cache only after its inclusion report lands.

#### Scenario: output phase while the topology republish is queued
- **WHEN** a charger with no cached grid type reports a grid type and then an output phase before the
  topology inclusion report has been published
- **THEN** the leg is mapped for the new topology and `sup_phase_modes` is the settable set of that topology
