## MODIFIED Requirements

### Requirement: Grid Type Observation Refreshes The Specification
A detected-power-grid-type observation SHALL first publish grounding-fault and grid-type-fault alarm
reports, before the equivalence check below, because several raw grid types map onto the same FIMP
pair and a fault must be reported even when the topology is unchanged. An observation older than the
cached grid type, or than the newest one already observed (changed, unchanged or still awaiting
publication), SHALL then stop the handler, because the cache would reject it after the props took
it. When the mapped grid type and phase count both equal the cached values the handler SHALL then
stop. Otherwise the installation parameters SHALL be cached, the chargepoint service specification
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
