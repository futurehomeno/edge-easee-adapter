## ADDED Requirements

### Requirement: Output Phase Uses The Advertised Topology
The output-phase handler SHALL derive `sup_phase_modes` from the grid type and phase count in the
chargepoint specification, not from the cache, because a topology republish replaces the
specification at once but commits the cache only after its inclusion report lands.

#### Scenario: output phase while the topology republish is queued
- **WHEN** a charger with no cached grid type reports a grid type and then an output phase before the
  topology inclusion report has been published
- **THEN** `sup_phase_modes` advertises the reported leg for the new topology
