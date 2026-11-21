## MODIFIED Requirements
### Requirement: Power Observation
A total-power observation SHALL be cached in watts, converting from the kilowatts Easee reports by
multiplying by 1000. A successful cache update SHALL publish an unforced `W` meter report followed by
an unforced extended report for `power_import`, and then an unforced chargepoint state report, so a
state change that the power reading reveals is published without waiting for the next state
observation. The state report SHALL be attempted even when a meter report fails, and SHALL NOT be
sent until a charger state has been observed.

#### Scenario: power observation arrives
- **WHEN** a total-power observation of 7.4 arrives
- **THEN** 7400 is cached and the two unforced meter reports are published

#### Scenario: power drops to zero after a state masked by residual power
- **WHEN** `charging` was the last state report, an `await_start` state observation arrived while a
  power reading above zero not older than it was cached, and then a 0 W power observation arrives
- **THEN** one `evt.state.report` with `ready_to_charge` is published after the 0 W reading

#### Scenario: power reading without a state change
- **WHEN** a power observation arrives and the reported state is unchanged
- **THEN** no state report is published

#### Scenario: meter report fails
- **WHEN** `ready_to_charge` is cached, a 0 W power observation arrives and the `W` meter report
  fails
- **THEN** the unforced state report is still published and the meter report error is logged

#### Scenario: power reading before any charger state
- **WHEN** no charger state has been observed and a 0 W power observation arrives
- **THEN** the two meter reports are published and no state report
