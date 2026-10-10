# Observation Streaming

## ADDED Requirements

### Requirement: Every Report Leaves The Observation Drain
Every report published in response to an observation SHALL be handed to the handler's report queue,
alarm reports and inclusion reports included. The observation drain is the only drainer of the
observation channel, and every report takes a service lock a chargepoint command may hold for up to
`currentWaitDuration`; publishing inline parks the drain behind that command. A parked drain fills
the observation buffer, `ProductUpdate` then drops observations, and the session start/stop pair is
edge-triggered and never replayed - so a dropped one loses the session's DB and FIMP record.

#### Scenario: an alarm report while the drain is busy
- **WHEN** an error-code observation raises an alarm and the service lock is held
- **THEN** the report is queued and the observation drain continues

#### Scenario: an inclusion report while the drain is busy
- **WHEN** a detected-power-grid-type observation republishes the chargepoint props
- **THEN** the thing update and inclusion report are queued and the observation drain continues

### Requirement: Close Stops Every Publisher
Closing an observations handler SHALL stop every goroutine that can publish on its thing and SHALL
wait for each to finish, the lifetime-energy goroutine included. `Unregister` closes the handler, so
after `cmd.thing.delete` a goroutine left running would publish a meter report on a thing that has
been torn down. Close SHALL remain safe to call more than once and safe against an observation being
dispatched concurrently.

#### Scenario: close with the energy goroutine running
- **WHEN** a lifetime-energy observation has started the energy goroutine and the handler is closed
- **THEN** the goroutine stops without publishing, and Close returns only once it has finished

#### Scenario: close before any lifetime energy arrives
- **WHEN** the handler is closed and no lifetime-energy observation ever started the goroutine
- **THEN** Close returns normally

#### Scenario: close twice
- **WHEN** Close is called a second time
- **THEN** it returns without panicking
