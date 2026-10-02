## MODIFIED Requirements

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
SHALL be the advertised `sup_phase_modes` for the grid type, phase count and persisted phase, so the
report stays within what the inclusion report advertises instead of failing.

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
- **THEN** the persisted phase is reported when it is advertised, otherwise the first advertised
  mode, without an error

#### Scenario: nothing mappable
- **WHEN** the refetched state yields no supported modes
- **THEN** an error is logged with the charger ID, grid type, phases and internal mode, and
  `unable to map phase modes` is returned
