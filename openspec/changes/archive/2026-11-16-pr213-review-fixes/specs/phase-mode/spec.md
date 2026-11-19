## MODIFIED Requirements

### Requirement: Phase Mode Observation Handling
The phase-mode observation (Easee's internal mode) SHALL be written to the cache subject to the
timestamp guard, also when it equals the cached internal mode: `cmd.phase_mode.set` caches the sent
target with the timestamp of the observation it replaced, so its echo has to record its own, or an
older observation dispatched after it would replace the newer mode. An observation equal to the
cached internal mode SHALL publish nothing. Otherwise an accepted observation SHALL publish an
unforced `evt.phase_mode.report` on its own goroutine - the report takes the chargepoint service
lock that `cmd.phase_mode.set` holds while it waits for an observation, and the dispatch loop
publishing it is the only drainer of that observation.

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
