## MODIFIED Requirements

### Requirement: Phase Mode Observation Handling
The phase-mode observation (Easee's internal mode) SHALL be written to the cache subject to the
timestamp guard, also when it equals the cached internal mode: `cmd.phase_mode.set` caches the sent
target with the timestamp of the observation it replaced, so its echo has to record its own, or an
older observation dispatched after it would replace the newer mode. An observation equal to the
cached internal mode SHALL publish nothing. Otherwise an accepted observation SHALL publish an
unforced `evt.phase_mode.report` on its own goroutine - the report takes the chargepoint service
lock that `cmd.phase_mode.set` holds while it waits for an observation, and the dispatch loop
publishing it is the only drainer of that observation. The report SHALL be skipped, without an
error, when the chargepoint service advertises no `sup_phase_modes` (it is not phase-mode aware,
and cliffhanger refuses the report). The check SHALL run when the report is sent rather than when it
is queued, because a topology republish queued in between can change the props.

#### Scenario: internal mode changes
- **WHEN** a phase-mode observation carries a value different from the cached one
- **THEN** the cache is updated and an unforced phase-mode report is published

#### Scenario: internal mode unchanged
- **WHEN** the observation repeats the cached internal mode
- **THEN** nothing is published

#### Scenario: an older observation after a repeat
- **WHEN** the cache holds mode 1 stamped t0, an observation of mode 1 stamped t2 arrives, then one
  of mode 2 stamped t1, with t0 < t1 < t2
- **THEN** the cache holds mode 1 stamped t2 and nothing is published

#### Scenario: internal mode changes on a charger without phase modes
- **WHEN** a charger whose chargepoint props carry no `sup_phase_modes` (grid type 30, or a grid
  type not yet detected) reports an internal mode different from the cached one
- **THEN** the cache is updated, no phase-mode report is sent and no
  `aware phase modes is not supported` warning is logged

### Requirement: Output Phase Observation Handling
The output-phase observation SHALL be translated to a FIMP phase mode and cached. The translation
SHALL follow the FIMP grid type in the chargepoint props (replaced as soon as a topology republish is queued, ahead of the cache) rather than the TN/IT suffix of Easee's value, because a
charger on an IT grid has been seen reporting the TN variant: on an IT or TT grid a TN-suffixed
value SHALL map to the mode of its IT twin on the same terminals (10 as 11, 12 as 13, 20 as 22),
and on a TN grid an IT-suffixed value SHALL map to the mode of its TN twin. A value without a twin,
or any value while the grid type is unknown, SHALL map on its suffix. A translation
yielding an empty mode SHALL be dropped without updating the cache, because the charger reports an
unassigned output phase whenever it is not charging, even during an ongoing session. A successful
cache update SHALL publish an unforced `evt.phase_mode.report`, skipped as in Phase Mode Observation
Handling when the chargepoint service advertises no `sup_phase_modes`; the reported leg SHALL be
persisted either way.

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

#### Scenario: topology republish still queued
- **WHEN** the props already carry an IT grid but the cache still holds TN, and the charger reports `P1_T2_T3_TN` (10)
- **THEN** `L1L2` is cached and reported

#### Scenario: charging on a charger without phase modes
- **WHEN** a charger whose chargepoint props carry no `sup_phase_modes` reports a single leg newer
  than the cached one
- **THEN** the leg is cached and persisted, no phase-mode report is sent and no
  `aware phase modes is not supported` warning is logged
