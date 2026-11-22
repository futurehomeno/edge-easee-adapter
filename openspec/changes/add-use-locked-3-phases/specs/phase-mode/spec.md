## MODIFIED Requirements

### Requirement: FIMP To Easee Mode Mapping
A requested FIMP phase mode SHALL map to Easee internal mode 1 when it appears in the single-phase
row, and to internal mode 2 otherwise, provided it appears in the settable set. Three-phase requests
map to auto rather than the internal three-phase lock, because most EVs refuse a 1-to-3 phase
transition mid-session and would stall on a hard lock. When `use_locked_3_phases` is on and the
cached charger state is a session in progress (`ready_to_charge`, `charging` or `suspended_by_ev`),
a request that would map to internal mode 2 SHALL map to internal mode 3 instead; every such request
is the multi-phase mode, the only member of the internal mode 3 row. The same mapping SHALL decide
whether a recorded request still holds against a newer internal mode. A mode in neither set SHALL be
rejected with `phase modes mapper: mode <mode> unsupported on a <grid> grid with <n> phases`.

#### Scenario: single-leg request
- **WHEN** `NL1` is requested on a TN grid with 3 phases
- **THEN** Easee internal mode 1 is the target

#### Scenario: three-phase request
- **WHEN** `NL1L2L3` is requested on a TN grid with 3 phases and `use_locked_3_phases` is off
- **THEN** Easee internal mode 2 (auto) is the target, never internal mode 3

#### Scenario: three-phase request with the lock during a session
- **WHEN** `NL1L2L3` is requested on a TN grid with 3 phases, `use_locked_3_phases` is on and the
  charger is charging
- **THEN** Easee internal mode 3 is the target

#### Scenario: three-phase request with the lock and no session
- **WHEN** `NL1L2L3` is requested, `use_locked_3_phases` is on and the charger is disconnected
- **THEN** Easee internal mode 2 (auto) is the target

#### Scenario: single leg with the lock
- **WHEN** `NL2` is requested on a TN grid with 3 phases, `use_locked_3_phases` is on and a session
  is in progress
- **THEN** Easee internal mode 1 is the target

#### Scenario: unsupported mode
- **WHEN** a mode outside the matrix rows for that grid and phase count is requested
- **THEN** the mapping fails with the `unsupported on a <grid> grid` error

## ADDED Requirements

### Requirement: Three-Phase Lock Follows The Session
When `use_locked_3_phases` is on and a charger state observation changes the cached FIMP state to a
known one, the adapter SHALL request the advertised multi-phase mode through the chargepoint service
when a session started on cached internal mode 2, or a session ended on cached internal mode 3, so
the mapping locks or releases three phases for the new state. At the first such change after the
setting went off, the same request SHALL be made on cached internal mode 3 whatever the state, so the
lock does not outlive the setting; a mode 3 found after that was set elsewhere and SHALL be left
alone. The request SHALL be queued after the state report, and SHALL be skipped when fewer than two
modes are advertised. Internal mode 1, an unknown or offline state, an unchanged state, or the
setting off since the previous change SHALL trigger nothing.

#### Scenario: session starts on auto
- **WHEN** the setting is on, the internal mode is 2 and the state changes from `disconnected` to
  `ready_to_charge`
- **THEN** the three-phase mode is requested and Easee internal mode 3 is sent

#### Scenario: car disconnected on the lock
- **WHEN** the setting is on, the internal mode is 3 and the state changes to `disconnected`
- **THEN** the three-phase mode is requested and Easee internal mode 2 is sent

#### Scenario: session completed on the lock
- **WHEN** the setting is on, the internal mode is 3 and the state changes to `finished`
- **THEN** the three-phase mode is requested and Easee internal mode 2 is sent

#### Scenario: single phase is left alone
- **WHEN** the setting is on, the internal mode is 1 and a session starts or ends
- **THEN** no phase mode is requested

#### Scenario: unknown state
- **WHEN** the setting is on, the internal mode is 3 and the charger goes offline
- **THEN** no phase mode is requested

#### Scenario: setting off
- **WHEN** the setting is off and a session starts on internal mode 2
- **THEN** no phase mode is requested

#### Scenario: setting turned off during a session
- **WHEN** the setting goes off while the internal mode is 3, and the state then changes from
  `charging` to `suspended_by_ev`
- **THEN** the three-phase mode is requested and Easee internal mode 2 is sent
- **AND** a later state change on internal mode 3 requests nothing
