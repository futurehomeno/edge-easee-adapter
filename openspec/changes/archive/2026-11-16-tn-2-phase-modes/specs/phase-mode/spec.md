## MODIFIED Requirements

### Requirement: Phase Mode Matrix
Supported FIMP phase modes SHALL be derived from a matrix keyed by grid type (TN, TT, IT), phase
count (1 or 3, and 2 on TN) and Easee internal phase mode (1, 2 or 3). A TN grid uses
neutral-referenced legs (`NL1`, `NL2`, `NL3`, `NL1L2`, `NL1L2L3`); TT and IT grids use line-to-line
legs (`L1L2`, `L2L3`, `L3L1`, `L1L2L3`). A 1-phase installation SHALL yield exactly one mode — `NL1`
on TN, `L1L2` on TT and IT — for internal modes 1 and 2; internal mode 3 is not defined for a
1-phase installation. A TN 2-phase installation SHALL use the legs of Easee's nominal 2-phase
wiring (neutral on pin 2, L1 on pin 3, L2 on pin 4): `NL1` and `NL2` for internal mode 1, `NL1`,
`NL2` and `NL1L2` for internal mode 2, and `NL1L2` for internal mode 3.
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

#### Scenario: TN grid, two phases
- **WHEN** modes are requested for grid TN, internal mode 2, 2 phases
- **THEN** `NL1`, `NL2` and `NL1L2` are returned and nothing is logged as an error

#### Scenario: TN grid, two phases, reported leg
- **WHEN** a TN 2-phase charger reports `NL2` and `NL2` is requested
- **THEN** Easee internal mode 1 is the target, and `NL1L2` maps to internal mode 2

#### Scenario: grid type not yet detected
- **WHEN** the grid type is empty, or the phase count or internal mode is zero
- **THEN** no modes are returned and nothing is logged as an error

#### Scenario: unknown grid type
- **WHEN** modes are requested for a grid type absent from the matrix
- **THEN** an error is logged and no modes are returned
