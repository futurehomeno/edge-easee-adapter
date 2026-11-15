## ADDED Requirements

### Requirement: Advertised Modes Do Not Follow The Observed Leg
`sup_phase_modes` SHALL stay the settable set for the charger's grid type and phase count when the
charger reports a single leg, because an Easee picks its leg per session and a list narrowed to the
last leg makes cliffhanger reject a request for another one. The reported leg SHALL be persisted for
the phase-mode report without republishing the inclusion report. The write SHALL go with the queued
phase-mode report, not run on the observation drain, and a failed write SHALL NOT hold the report back.

#### Scenario: an IT charger changes leg
- **GIVEN** a charger on an IT grid with 3 phases advertising `L1L2`, `L2L3`, `L3L1` and `L1L2L3`, last seen on `L1L2`
- **WHEN** it reports output phase `P1_T2_T4_IT` (`L3L1`)
- **THEN** all four modes stay advertised, `L3L1` is persisted and no inclusion report is sent

#### Scenario: the leg cannot be saved
- **WHEN** a charger reports a new leg and writing it to the thing state blocks or fails
- **THEN** the observation is handled without waiting for the write and the phase-mode report is still published

## MODIFIED Requirements

### Requirement: Output Phase Uses The Advertised Topology
The output-phase handler SHALL map the reported leg with the grid type in the chargepoint
specification, not the cache, because a topology republish replaces the specification at once but
commits the cache only after its inclusion report lands.

#### Scenario: output phase while the topology republish is queued
- **WHEN** a charger with no cached grid type reports a grid type and then an output phase before the
  topology inclusion report has been published
- **THEN** the leg is mapped for the new topology and `sup_phase_modes` is the settable set of that topology

### Requirement: Phase Mode Report Precedence
`evt.phase_mode.report` SHALL resolve the reported mode in this order: a mode the adapter itself
requested, when one is cached, its timestamp is later than the cached output phase's timestamp and
the request still holds; otherwise the cached output phase, when it is not empty and not stale;
otherwise a mode derived from a freshly fetched charger state. The requested mode outranks the
output phase because the output phase goes unassigned between sessions and that observation is
dropped, leaving the cached value a stale echo of the previous session.

A cached request SHALL stop holding once an internal phase-mode observation newer than the request
maps to an Easee mode other than the one the request asked for, or the request no longer maps to an
Easee mode at all: the mode was changed elsewhere, and nothing else ever clears the request.

The fallback SHALL be the last supported mode when the charger's internal mode is auto and the
first otherwise: the auto row ends with the multi-phase mode, which is what a three-phase request
maps onto, so reporting the first would answer a request the user just made with a single leg.
A persisted phase among the supported modes SHALL be preferred over the first.

When the refetched internal mode is 0 (Easee leaves it unset on some models) the supported modes
SHALL be the settable modes for the grid type and phase count, whatever leg is persisted, so the
report stays within what the inclusion report advertises instead of failing. An unset mode SHALL be
read as auto and report the last advertised mode: a single leg would let energy guard load the other
two phases while the charger may draw on all three.

A cached output phase SHALL count as stale once an internal phase-mode observation newer than it no
longer lists that leg among the supported modes — unless the charger is charging, because Easee
applies a new mode only at a session boundary and the leg in use is still the old one.

#### Scenario: recent request outranks an older output phase
- **WHEN** a requested mode is cached with a timestamp later than the cached output phase and no
  newer internal mode contradicts it
- **THEN** the requested mode is reported

#### Scenario: output phase is newer
- **WHEN** the cached output phase carries a later timestamp than the requested mode and no newer
  internal mode voids that leg
- **THEN** the output phase is reported

#### Scenario: the charger left the requested mode
- **WHEN** an internal phase-mode observation newer than the cached request maps to a different
  Easee mode than the request asked for
- **THEN** the request is passed over and resolution continues with the cached output phase

#### Scenario: internal mode voids an idle leg
- **WHEN** an internal phase-mode observation newer than the cached output phase no longer covers
  that leg and the charger is not charging
- **THEN** the cached output phase is passed over and the refetched state's fallback mode is
  reported

#### Scenario: a charging charger keeps its leg
- **WHEN** the same newer internal mode arrives while the charger is charging
- **THEN** the cached output phase is still reported

#### Scenario: nothing cached
- **WHEN** neither a requested mode nor an output phase is cached and the charger is not in auto
- **THEN** the charger state is refetched and the first supported mode is reported

#### Scenario: nothing cached on an auto charger
- **WHEN** neither a requested mode nor an output phase is cached and the charger sits in auto
- **THEN** the charger state is refetched and the last supported mode, the multi-phase one, is
  reported

#### Scenario: internal mode unset on a 1-phase charger
- **WHEN** neither a requested mode nor an output phase is cached and the refetched state carries
  grid TN, 1 phase and internal mode 0
- **THEN** `NL1` is reported without an error

#### Scenario: internal mode unset on a 3-phase charger
- **WHEN** neither a requested mode nor an output phase is cached and the refetched state carries
  grid TN, 3 phases and internal mode 0
- **THEN** `NL1L2L3` is reported without an error, whatever leg is persisted

#### Scenario: nothing mappable
- **WHEN** the refetched state yields no supported modes
- **THEN** an error is logged with the charger ID, grid type, phases and internal mode, and
  `unable to map phase modes` is returned
