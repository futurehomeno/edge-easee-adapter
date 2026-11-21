## MODIFIED Requirements

### Requirement: Settable Phase Modes
The modes advertised as settable SHALL be the auto row of the matrix for the charger's grid type and
phase count, which is the union of the other rows. Easee grid type 30 (`WarningTN2PhasePin235`)
SHALL advertise none: it wires neutral on pin 2, L1 on pin 3 and L3 on pin 5, so the TN 2-phase
row's `NL2` and `NL1L2` are out of its reach, and FIMP has no `NL1L3` to name its two-phase mode.
Its FIMP grid type and phase count SHALL stay TN with 2 phases, and the grid-type-fault alarm is its
report. The other TN 2-phase grid types - 2, 36 (`WarningTN2PhasePIN234`, the nominal pins 2, 3, 4)
and 38 (`WarningTN2PhaseGNDFault`, a fault on the ground, not on the phase pins) - SHALL keep the
row. The persisted charger state SHALL carry Easee's raw grid type next to the FIMP grid type and
phase count; a state without it, stored by an older adapter, SHALL advertise the row of its FIMP
grid type and phase count. The modes SHALL be published as `sup_phase_modes` on the chargepoint
service, and only when the list is non-empty.

#### Scenario: TN grid, three phases
- **WHEN** the settable modes are computed for grid TN with 3 phases
- **THEN** `NL1`, `NL2`, `NL3` and `NL1L2L3` are advertised

#### Scenario: grid type unknown at thing creation
- **WHEN** the charger state carries no grid type
- **THEN** `sup_phase_modes` is not added to the chargepoint specification

#### Scenario: TN 2-phase on pins 2, 3, 5 at thing creation
- **WHEN** a charger whose config reports Easee grid type 30 is created
- **THEN** `sup_phase_modes` is not added, grid type TN with 2 phases is advertised, and the state
  is persisted with raw grid type 30

#### Scenario: other TN 2-phase grid types at thing creation
- **WHEN** a charger whose config reports Easee grid type 2, 36 or 38 is created
- **THEN** `NL1`, `NL2` and `NL1L2` are advertised

#### Scenario: state stored by an older adapter
- **WHEN** the charger config cannot be read and the stored state carries grid TN with 2 phases but
  no raw grid type
- **THEN** `NL1`, `NL2` and `NL1L2` are advertised

#### Scenario: stored state on pins 2, 3, 5
- **WHEN** the charger config cannot be read and the stored state carries raw grid type 30
- **THEN** `sup_phase_modes` is not added

### Requirement: Grid Type Observation Refreshes The Specification
A detected-power-grid-type observation SHALL first publish grounding-fault and grid-type-fault alarm
reports, before the equivalence check below, because several raw grid types map onto the same FIMP
pair and a fault must be reported even when the topology is unchanged. An observation older than the
cached grid type, or than the newest one already observed (changed, unchanged or still awaiting
publication), SHALL then stop the handler, because the cache would reject it after the props took
it. When the mapped grid type and phase count both equal the cached values, and the settable modes
of the observed raw grid type equal those of the raw grid type last advertised, the handler SHALL
then stop: raw grid types 2 and 30 share TN with 2 phases but not their settable modes. The raw grid
type last advertised SHALL be seeded from the persisted state the thing was created from, so a
normal start republishes nothing for an unchanged charger, and SHALL be committed with the cache
once a republish has landed, so a dropped republish is retried by the next observation. A state
without a raw grid type SHALL count as advertising the row of the cached grid type and phase count.
Otherwise the installation parameters SHALL be cached, the chargepoint service specification
refreshed and a thing inclusion report published, so that the new phase-mode and grid-type
properties reach consumers.

#### Scenario: fault on an unchanged topology
- **WHEN** a grid type mapping to the same FIMP grid and phase count but carrying a fault arrives
- **THEN** the grounding-fault and grid-type-fault alarms are reported before the handler stops

#### Scenario: outdated topology
- **WHEN** a grid type differing from the cache arrives with a timestamp older than the cached one
- **THEN** the props, the cache and the inclusion report are left untouched

#### Scenario: older than a queued topology
- **WHEN** a grid type arrives older than a newer one whose republish is still queued
- **THEN** the props keep the newer topology and no second inclusion report is sent

#### Scenario: older than an unchanged topology
- **WHEN** a grid type arrives older than a newer one that matched the cache
- **THEN** the props, the cache and the inclusion report are left untouched

#### Scenario: topology changes
- **WHEN** the mapped grid type or phase count differs from the cache
- **THEN** the installation parameters are cached, the chargepoint specification is refreshed and an
  inclusion report is sent

#### Scenario: a new topology on pins 2, 3, 5
- **WHEN** a charger cached as TN with 3 phases reports Easee grid type 30
- **THEN** grid type TN with 2 phases is advertised without `sup_phase_modes` and an inclusion
  report is sent

#### Scenario: pins 2, 3, 5 appear at runtime
- **WHEN** a charger created on Easee grid type 2 reports grid type 30
- **THEN** `sup_phase_modes` is removed, TN with 2 phases stays advertised and an inclusion report
  is sent

#### Scenario: pins 2, 3, 5 cleared at runtime
- **WHEN** a charger created on Easee grid type 30 reports grid type 2
- **THEN** `NL1`, `NL2` and `NL1L2` are advertised and an inclusion report is sent

#### Scenario: unchanged raw grid type on start
- **WHEN** a charger created on Easee grid type 30, or on 2, reports the same grid type
- **THEN** no inclusion report is sent

#### Scenario: another TN 2-phase grid type on the same row
- **WHEN** a charger created on Easee grid type 2 reports grid type 36 or 38
- **THEN** no inclusion report is sent

#### Scenario: state stored by an older adapter on pins 2, 3, 5
- **WHEN** a charger created from a state without a raw grid type, cached as TN with 2 phases,
  reports Easee grid type 30
- **THEN** `sup_phase_modes` is removed and an inclusion report is sent

#### Scenario: dropped republish of pins 2, 3, 5
- **WHEN** the inclusion report for a change from grid type 2 to 30 fails and grid type 30 is
  reported again
- **THEN** the inclusion report is sent again
