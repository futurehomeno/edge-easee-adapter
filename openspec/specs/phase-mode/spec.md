# Phase Mode Specification

## Purpose
Map between FIMP phase modes (which name concrete legs, such as `NL1` or `NL1L2L3`) and Easee's
internal phase mode (an integer: 1 single-phase lock, 2 auto, 3 three-phase lock), advertise which
modes a charger supports, apply a requested mode, and report the mode currently in effect.

## Requirements

### Requirement: Phase Mode Matrix
Supported FIMP phase modes SHALL be derived from a matrix keyed by grid type (TN, TT, IT), phase
count (1 or 3) and Easee internal phase mode (1, 2 or 3). A TN grid uses neutral-referenced legs
(`NL1`, `NL2`, `NL3`, `NL1L2L3`); TT and IT grids use line-to-line legs (`L1L2`, `L2L3`, `L3L1`,
`L1L2L3`). A 1-phase installation SHALL yield exactly one mode — `NL1` on TN, `L1L2` on TT and IT —
for internal modes 1 and 2; internal mode 3 is not defined for a 1-phase installation.
An empty grid type, a zero internal mode or a zero phase count SHALL yield no modes.
An unknown grid type, phase count or internal mode SHALL be logged as an error and yield no modes.

#### Scenario: TN grid, three phases, single-phase lock
- **WHEN** modes are requested for grid TN, internal mode 1, 3 phases
- **THEN** `NL1`, `NL2` and `NL3` are returned

#### Scenario: TN grid, three phases, auto
- **WHEN** modes are requested for grid TN, internal mode 2, 3 phases
- **THEN** `NL1`, `NL2`, `NL3` and `NL1L2L3` are returned

#### Scenario: TT grid, three phases, three-phase lock
- **WHEN** modes are requested for grid TT, internal mode 3, 3 phases
- **THEN** `L1L2L3` alone is returned

#### Scenario: TN grid, one phase
- **WHEN** modes are requested for grid TN, internal mode 1 or 2, 1 phase
- **THEN** `NL1` alone is returned

#### Scenario: IT grid, one phase
- **WHEN** modes are requested for grid IT, internal mode 1 or 2, 1 phase
- **THEN** `L1L2` alone is returned

#### Scenario: grid type not yet detected
- **WHEN** the grid type is empty, or the phase count or internal mode is zero
- **THEN** no modes are returned and nothing is logged as an error

#### Scenario: unknown grid type
- **WHEN** modes are requested for a grid type absent from the matrix
- **THEN** an error is logged and no modes are returned

### Requirement: Settable Phase Modes
The modes advertised as settable SHALL be the auto row of the matrix for the charger's grid type and
phase count, which is the union of the other rows. They SHALL be published as `sup_phase_modes` on the
chargepoint service, and only when the list is non-empty.

#### Scenario: TN grid, three phases
- **WHEN** the settable modes are computed for grid TN with 3 phases
- **THEN** `NL1`, `NL2`, `NL3` and `NL1L2L3` are advertised

#### Scenario: grid type unknown at thing creation
- **WHEN** the charger state carries no grid type
- **THEN** `sup_phase_modes` is not added to the chargepoint specification

### Requirement: FIMP To Easee Mode Mapping
A requested FIMP phase mode SHALL map to Easee internal mode 1 when it appears in the single-phase
row, and to internal mode 2 otherwise, provided it appears in the settable set. Three-phase requests
map to auto rather than the internal three-phase lock, because most EVs refuse a 1-to-3 phase
transition mid-session and would stall on a hard lock. A mode in neither set SHALL be rejected with
`phase modes mapper: mode <mode> unsupported on a <grid> grid with <n> phases`.

#### Scenario: single-leg request
- **WHEN** `NL1` is requested on a TN grid with 3 phases
- **THEN** Easee internal mode 1 is the target

#### Scenario: three-phase request
- **WHEN** `NL1L2L3` is requested on a TN grid with 3 phases
- **THEN** Easee internal mode 2 (auto) is the target, never internal mode 3

#### Scenario: unsupported mode
- **WHEN** a mode outside the matrix rows for that grid and phase count is requested
- **THEN** the mapping fails with the `unsupported on a <grid> grid` error

### Requirement: Applying A Phase Mode
`cmd.phase_mode.set` SHALL map the requested mode to an Easee internal mode and, when that target
differs from the cached internal mode, post it to `/api/chargers/{id}/commands/set_phase_mode`, then
record the requested FIMP mode in the cache, then restart an in-progress session. The request SHALL
be stamped one millisecond past the newer of the cached output-phase and internal-mode timestamps
rather than with the hub clock, because those Easee timestamps are the only values it is ever
compared against. A grid with fewer than two settable modes SHALL make the command a no-op:
flipping the internal mode would bounce a session for a change nothing can observe. When the target
equals the cached internal mode the command SHALL return successfully without calling Easee and
without recording a requested mode, so that a fresh timestamp cannot outrank a live output-phase
observation.

#### Scenario: mode changes
- **WHEN** the target internal mode differs from the cached one
- **THEN** Easee is called, the requested mode is cached ahead of the newest observation, and the
  session restart runs

#### Scenario: hub clock behind Easee
- **WHEN** the cached observations carry timestamps ahead of the hub clock
- **THEN** the requested mode is still stamped past them and outranks them in the next report

#### Scenario: only one settable mode
- **WHEN** the grid and phase count leave a single settable mode
- **THEN** no Easee call is made, no requested mode is recorded, and the command reports success

#### Scenario: target already in effect
- **WHEN** the target internal mode equals the cached internal mode
- **THEN** no Easee call is made, no requested mode is recorded, and the command reports success

#### Scenario: Easee rejects the change
- **WHEN** the set-phase-mode call fails
- **THEN** the error is returned and no requested mode is cached

### Requirement: Session Restart To Apply A Mode
Because the charger applies a new phase mode only at a session boundary, an in-progress session SHALL
be bounced after the mode is set. The restart SHALL be skipped when the charger state is not
charging. A failure to read the state, or a failure to pause, SHALL be logged as a warning and
treated as success — the mode is stored and takes effect on the next session anyway. A failure to
resume SHALL be returned as `phase mode set to <target>, but the charger was left stopped`, and a
resume Easee accepts but the charger never echoes back within the current wait duration as
`phase mode set to <target>, but the charger did not resume at <current>A` - an unconfirmed resume
may well have left the session paused. The resume SHALL restore the session's own current — the
cached requested offered current, falling back to the cached max current when that is zero or less
— read before the pause, since the
session-finished observation clears it asynchronously. It SHALL NOT be issued as a normal-mode
start, which would floor the current to `initial_charging_current` and silently raise a slow
session; the session's mode is recorded nowhere, so it cannot be reconstructed.

#### Scenario: charger idle
- **WHEN** the charger state is not charging
- **THEN** no stop or start is issued

#### Scenario: a slow session is bounced
- **WHEN** a session running below `initial_charging_current` is bounced to apply a mode
- **THEN** it resumes at the current it was running, not at `initial_charging_current`

#### Scenario: pause fails
- **WHEN** stopping the session fails
- **THEN** a warning is logged and the command reports success

#### Scenario: resume fails
- **WHEN** the session was paused but the restart fails
- **THEN** the command fails with `the charger was left stopped`

#### Scenario: resume is never confirmed
- **WHEN** Easee accepts the resume but no observation echoes the current back in time
- **THEN** the command fails with `the charger did not resume`

#### Scenario: charger state unknown
- **WHEN** the state report fails
- **THEN** a warning is logged and the command reports success

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

#### Scenario: nothing mappable
- **WHEN** the refetched state yields no supported modes
- **THEN** an error is logged with the charger ID, grid type, phases and internal mode, and
  `unable to map phase modes` is returned

### Requirement: Phase Mode Observation Handling
The phase-mode observation (Easee's internal mode) SHALL be ignored when it equals the cached
internal mode. Otherwise the cache SHALL be updated, subject to the timestamp guard, and an
unforced `evt.phase_mode.report` published on its own goroutine - the report takes the chargepoint
service lock that `cmd.phase_mode.set` holds while it waits for an observation, and the dispatch
loop publishing it is the only drainer of that observation.

#### Scenario: internal mode changes
- **WHEN** a phase-mode observation carries a value different from the cached one
- **THEN** the cache is updated and an unforced phase-mode report is published

#### Scenario: internal mode unchanged
- **WHEN** the observation repeats the cached internal mode
- **THEN** nothing is published

### Requirement: Output Phase Observation Handling
The output-phase observation SHALL be translated to a FIMP phase mode and cached. A translation
yielding an empty mode SHALL be dropped without updating the cache, because the charger reports an
unassigned output phase whenever it is not charging, even during an ongoing session. A successful
cache update SHALL publish an unforced `evt.phase_mode.report`.

#### Scenario: charger not charging
- **WHEN** the output-phase observation maps to an empty mode
- **THEN** the observation is dropped and the cached output phase is preserved

#### Scenario: charging on a known leg
- **WHEN** the output-phase observation maps to a concrete mode newer than the cached one
- **THEN** the cache is updated and an unforced phase-mode report is published

### Requirement: Grid Type Observation Refreshes The Specification
A detected-power-grid-type observation SHALL first publish grounding-fault and grid-type-fault alarm
reports, before the equivalence check below, because several raw grid types map onto the same FIMP
pair and a fault must be reported even when the topology is unchanged. When the mapped grid type and
phase count both equal the cached values the handler SHALL then stop. Otherwise the installation
parameters SHALL be cached, the chargepoint service specification refreshed and a thing inclusion
report published, so that the new phase-mode and grid-type properties reach consumers.

#### Scenario: fault on an unchanged topology
- **WHEN** a grid type mapping to the same FIMP grid and phase count but carrying a fault arrives
- **THEN** the grounding-fault and grid-type-fault alarms are reported before the handler stops

#### Scenario: topology changes
- **WHEN** the mapped grid type or phase count differs from the cache
- **THEN** the installation parameters are cached, the chargepoint specification is refreshed and an
  inclusion report is sent
