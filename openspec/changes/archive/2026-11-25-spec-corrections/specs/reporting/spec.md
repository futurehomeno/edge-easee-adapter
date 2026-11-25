## MODIFIED Requirements

### Requirement: Periodic Reporting Task
A car-charger reporting task SHALL run on `polling_interval` (default 20m) and SHALL be gated on the
app being connected, so that polling stops while the app reports a disconnected state.

#### Scenario: app connected
- **WHEN** the polling interval elapses and the lifecycle reports connected
- **THEN** the chargepoint, meter and alarm reports are refreshed

#### Scenario: app disconnected
- **WHEN** the lifecycle reports disconnected
- **THEN** the reporting task does not run
