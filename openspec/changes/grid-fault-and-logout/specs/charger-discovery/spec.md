# Charger Discovery

## ADDED Requirements

### Requirement: A Wiring Fault Does Not Replace The Stored Topology
A charger-config refresh whose `DetectedPowerGridType` yields zero phases SHALL leave the grid type,
the phase count and the phase mode at their stored values rather than writing the fault reading into
the thing state. Easee reports an undetected or mis-wired grid as grid type 50, 51 or 52, which map
to `("", 0)`, `(TN, 0)` and `(IT, 0)`; persisting one deletes `phases` and `sup_phase_modes` from
the chargepoint props, so cliffhanger rejects every `cmd.phase_mode.set` and energy guard drops the
charger from phase balancing. Because the state is what the next thing creation reads, a restart
during a fault would otherwise outlive the fault itself. The zero-phase alarm remains the report for
this condition, and a later healthy reading SHALL be applied normally.

#### Scenario: a wiring fault is fetched at thing creation
- **WHEN** the charger config reports a grid type that maps to zero phases
- **THEN** the stored grid type, phase count and phase mode are kept and nothing is persisted from
  that reading

#### Scenario: the fault clears
- **WHEN** a later charger config reports a grid type with a non-zero phase count
- **THEN** it replaces the stored topology as usual

#### Scenario: no topology has ever been stored
- **WHEN** a wiring fault is the first reading a charger ever returns
- **THEN** the state keeps its zero value, exactly as before the refresh, and the thing is created
  with the supported-max-current fallback
