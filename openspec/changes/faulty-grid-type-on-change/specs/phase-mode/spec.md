## ADDED Requirements

### Requirement: Faulty Grid Type Warning
The adapter SHALL log a warning with the charger ID when a charger's detected-power-grid-type
observation changes to a wiring warning, a wiring error or a value Easee has not documented, and
SHALL NOT log it again while the value is unchanged. Mapping a grid type, including the REST charger
config read on every idle phase-mode report, SHALL NOT log, because a lasting fault would otherwise
be logged on every poll.

#### Scenario: repeated faulty grid type
- **WHEN** a charger reports the same faulty grid type several times
- **THEN** one warning naming the charger and the grid type is logged

#### Scenario: faulty grid type returns
- **WHEN** a charger reports a faulty grid type, then a healthy one, then the faulty one again
- **THEN** a warning is logged for each change to the faulty grid type and none for the healthy one

#### Scenario: stale healthy grid type
- **WHEN** a charger reports a faulty grid type, then a healthy one older than that report, then the
  faulty one again
- **THEN** one warning is logged

#### Scenario: idle phase-mode report
- **WHEN** the phase-mode report reads the charger config of an idle charger with a faulty grid type
- **THEN** no grid type warning is logged
