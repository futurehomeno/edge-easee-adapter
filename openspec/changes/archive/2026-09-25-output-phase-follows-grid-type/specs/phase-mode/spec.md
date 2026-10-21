## MODIFIED Requirements

### Requirement: Output Phase Observation Handling
The output-phase observation SHALL be translated to a FIMP phase mode and cached. The translation
SHALL follow the cached FIMP grid type rather than the TN/IT suffix of Easee's value, because a
charger on an IT grid has been seen reporting the TN variant: on an IT or TT grid a TN-suffixed
value SHALL map to the mode of its IT twin on the same terminals (10 as 11, 12 as 13, 20 as 22),
and on a TN grid an IT-suffixed value SHALL map to the mode of its TN twin. A value without a twin,
or any value while the grid type is unknown, SHALL map on its suffix. A translation
yielding an empty mode SHALL be dropped without updating the cache, because the charger reports an
unassigned output phase whenever it is not charging, even during an ongoing session. A successful
cache update SHALL publish an unforced `evt.phase_mode.report`.

#### Scenario: charger not charging
- **WHEN** the output-phase observation maps to an empty mode
- **THEN** the observation is dropped and the cached output phase is preserved

#### Scenario: charging on a known leg
- **WHEN** the output-phase observation maps to a concrete mode newer than the cached one
- **THEN** the cache is updated and an unforced phase-mode report is published

#### Scenario: TN-suffixed phase on an IT grid
- **WHEN** a charger whose grid type is IT reports `P1_T2_T3_TN` (10)
- **THEN** `L1L2` is cached and reported, never `NL1`

#### Scenario: IT-suffixed phase on a TN grid
- **WHEN** a charger whose grid type is TN reports `P1_T2_T3_IT` (11)
- **THEN** `NL1` is cached and reported

#### Scenario: grid type not yet known
- **WHEN** the grid type is empty
- **THEN** the value maps on its own suffix
