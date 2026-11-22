## ADDED Requirements

### Requirement: Sub-Minimum Current Pauses
A `cmd.current_session.set_current` below `min_charging_current` (default 6 A), 0 A included, SHALL
pause the charger with the Easee pause-charging command, through the per-charger command channel
like `cmd.charge.stop`, and the adapter SHALL remember that pause as its own. The next value at or
above the minimum SHALL be written as the dynamic current with the recent-value dedup bypassed, which
resumes the charger, and SHALL clear that memory. A `cmd.charge.stop` SHALL
clear it too, so the adapter never adds a resume over a user stop.

The framework rejects a current under 6 A before the controller runs, so a value from 0 to 5 A SHALL
be handed to the controller in routing before the framework's handler: on success the message
interface SHALL be renamed with a `.paused` suffix so the framework's handler skips it, and the
adapter SHALL send the current-session report itself. A value from 6 A up to the minimum reaches the
controller through the framework and pauses there. A max current is still checked against 6 A.

#### Scenario: an energy manager sets 0 A
- **WHEN** `cmd.current_session.set_current` arrives with a value of 0
- **THEN** the pause-charging command is sent and no dynamic current is written
- **AND** the message interface is renamed with a `.paused` suffix and a current-session report is sent

#### Scenario: a current below a raised minimum
- **WHEN** `min_charging_current` is 10 and `cmd.current_session.set_current` arrives with 8
- **THEN** the charger is paused and no dynamic current is written

#### Scenario: the next valid value resumes the adapter's pause
- **WHEN** the adapter paused the charger for a sub-minimum value and a value at or above the minimum
  arrives, equal to the last one written inside `offered_current_wait_time`
- **THEN** the dynamic current is written anyway, resuming the charger

#### Scenario: a user stop is not resumed by the adapter
- **WHEN** the adapter paused the charger, then `cmd.charge.stop` arrives, then a value at or above
  the minimum equal to the last one written inside `offered_current_wait_time`
- **THEN** the dedup applies as for any write and nothing is sent

#### Scenario: a current under 6 A at a lowered minimum
- **WHEN** `min_charging_current` is 1 and `cmd.current_session.set_current` arrives with 3
- **THEN** 3 A is written as the dynamic current and the message interface is renamed with a `.paused` suffix

#### Scenario: the pause fails
- **WHEN** handing a value under 6 A to the controller returns an error
- **THEN** the message is left unrenamed and the framework handles it normally

#### Scenario: a negative value
- **WHEN** the value is below zero
- **THEN** the message is left untouched
