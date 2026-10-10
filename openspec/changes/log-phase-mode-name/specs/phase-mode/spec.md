## ADDED Requirements

### Requirement: Internal Phase Mode Naming
Easee's internal phase modes SHALL have a human-readable name: internal mode 1 is `single phase`,
internal mode 2 is `auto` and internal mode 3 is `three phase`. Any other value SHALL be named
`unknown`, so a mode Easee adds later is never given a wrong name.

#### Scenario: known internal mode
- **WHEN** a name is requested for internal mode 1, 2 or 3
- **THEN** `single phase`, `auto` and `three phase` are returned respectively

#### Scenario: unrecognised internal mode
- **WHEN** a name is requested for a value outside 1, 2 and 3
- **THEN** `unknown` is returned

### Requirement: Internal Phase Mode Logging
Every log line carrying an Easee internal phase mode SHALL name it and keep the number in
parentheses, as `<name> (<n>)`. The number is kept because it is the value on the wire, and a line
carrying only the name could not be matched against Easee's API.

The set-phase-mode call SHALL log `Set phase mode to <name> (<n>)`. The phase-mode observation
SHALL log `Phase mode=<name> (<n>)`; it previously read `Connected phases=<n>`, which named the
wrong quantity - the value is Easee's internal mode, not a count of connected phases.

#### Scenario: three-phase request logged
- **WHEN** internal mode 2 is posted for a charger
- **THEN** the line reads `Set phase mode to auto (2)`

#### Scenario: single-phase request logged
- **WHEN** internal mode 1 is posted for a charger
- **THEN** the line reads `Set phase mode to single phase (1)`

#### Scenario: the observation is logged
- **WHEN** a phase-mode observation carrying internal mode 2 arrives
- **THEN** the line reads `Phase mode=auto (2)`, not `Connected phases=2`
