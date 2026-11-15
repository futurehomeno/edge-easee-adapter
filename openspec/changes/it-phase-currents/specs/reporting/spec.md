## MODIFIED Requirements

### Requirement: Phase Current Observations
The in-current observations SHALL populate the phase 1, 2 and 3 currents from the terminals that
carry the lines on the grid type held in the chargepoint props: T3, T4 and T5 on a TN grid and while
the props carry no grid type, T2, T3 and T4 on an IT or TT grid. An in-current observation for a
terminal that carries no line on that grid (T2 on TN, T5 on IT or TT) SHALL be ignored: no cache
write, no report and no error. Each accepted observation SHALL publish its own unforced extended
meter report on a successful cache update.

#### Scenario: phase current observed
- **WHEN** an in-current T3 observation arrives newer than the cached value on a TN charger
- **THEN** the phase 1 current is cached and an unforced extended report is published

#### Scenario: IT or TT charger
- **WHEN** a charger whose props carry grid type IT or TT reports in-current T2, T3 and T4
- **THEN** they are cached as the phase 1, 2 and 3 currents, each with its own extended report

#### Scenario: terminal without a line
- **WHEN** a TN charger reports in-current T2, or an IT or TT charger reports in-current T5, even
  with a value that does not parse
- **THEN** no phase current is written, no report is published and the handler returns no error

#### Scenario: grid type unknown
- **WHEN** the props carry no grid type and in-current T2, T3, T4 and T5 arrive
- **THEN** T3, T4 and T5 are cached as the phase 1, 2 and 3 currents and T2 is ignored

#### Scenario: topology republish still pending
- **WHEN** the props already carry IT while the cache holds no or another grid type, and in-current
  T2 arrives
- **THEN** it is cached as the phase 1 current
