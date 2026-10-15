## MODIFIED Requirements

### Requirement: Cable Lock Parameter
The parameters service SHALL expose exactly one parameter, `cable_always_locked`, as a boolean select
with `Yes`/`No` options defaulting to false. `cmd.param.set` SHALL send the lock-state command to
Easee, treating both `200` and `202` as success, and, on success, seed the cache with the written value, because the framework answers the
command with a forced report the reporting cache cannot suppress. The seed SHALL carry the zero
timestamp rather than the hub's current time, so that the observation stays authoritative and
overwrites it whenever it lands: observations are stamped with Easee's clock, and a hub running
ahead of the server would otherwise have the cache's timestamp guard reject the real value.
`cmd.param.get_report` SHALL answer from the cache. Any other parameter ID SHALL be rejected
with `parameter: <id> not supported`.

#### Scenario: lock the cable
- **WHEN** `cable_always_locked` is set to true
- **THEN** the Easee lock-state command is sent with state true

#### Scenario: Easee answers 200
- **WHEN** the lock-state command is answered with `200 OK`
- **THEN** the set succeeds

#### Scenario: the observation contradicts the seed
- **WHEN** an observation carries a timestamp behind the hub's clock after a seed was written
- **THEN** the observed value still replaces the seed

#### Scenario: unknown parameter
- **WHEN** any other parameter ID is set or read
- **THEN** the call fails with `parameter: <id> not supported`

#### Scenario: charger echoes the lock setting
- **WHEN** the lock-cable-permanently observation arrives
- **THEN** the cache is updated and a forced parameter report is published
