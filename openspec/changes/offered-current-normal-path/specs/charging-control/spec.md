## MODIFIED Requirements

### Requirement: Sub-Minimum Current Pauses
A `cmd.current_session.set_current` below `min_charging_current` (default 6 A), 0 A included, SHALL
pause the charger with the Easee pause-charging command, through the per-charger command channel
like `cmd.charge.stop`, and the adapter SHALL remember that pause as its own. The next value at or
above the minimum SHALL be written as the dynamic current with the recent-value dedup bypassed, which
resumes the charger, and SHALL clear that memory. A `cmd.charge.stop` SHALL
clear it too: under a user stop no value resumes the charger (A User Stop Holds Until A Start).
Not met yet: two sub-minimum values handled at the same moment can leave the pause unremembered
when the first one's pause command fails (issue 40, open).

Every value, 0 A included, SHALL reach the controller through the framework's chargepoint handler,
which sends the current-session report on success and answers a failure with one error report. The
controller SHALL be called once per command. The framework checks a max current against 6 A.

#### Scenario: an energy manager sets 0 A
- **WHEN** `cmd.current_session.set_current` arrives with a value of 0
- **THEN** the pause-charging command is sent and no dynamic current is written
- **AND** exactly one current-session report is sent

#### Scenario: a current below a raised minimum
- **WHEN** `min_charging_current` is 10 and `cmd.current_session.set_current` arrives with 8
- **THEN** the charger is paused and no dynamic current is written

#### Scenario: the next valid value resumes the adapter's pause
- **WHEN** the adapter paused the charger for a sub-minimum value and a value at or above the minimum
  arrives, equal to the last one written inside `offered_current_wait_time`
- **THEN** the dynamic current is written anyway, resuming the charger

#### Scenario: a pause arrives while a valid value is being written
- **WHEN** a value at or above the minimum is still on its way to Easee and a sub-minimum value arrives
- **THEN** the pause stays the adapter's own once the write returns, and the next valid value resumes it

#### Scenario: a valid value arrives while the pause is being sent
- **WHEN** a sub-minimum value's pause is still on its way to Easee and a value at or above the
  minimum arrives, equal to the last one written inside `offered_current_wait_time`
- **THEN** the value is written anyway, resuming the charger

#### Scenario: a user stop is not resumed by the adapter
- **WHEN** the adapter paused the charger, then `cmd.charge.stop` arrives, then a value at or above
  the minimum equal to the last one written inside `offered_current_wait_time`
- **THEN** nothing is sent, and the value is kept only for the next `cmd.charge.start`

#### Scenario: a current under 6 A at a lowered minimum
- **WHEN** `min_charging_current` is 1 and `cmd.current_session.set_current` arrives with 3
- **THEN** 3 A is written as the dynamic current and a current-session report is sent

#### Scenario: the pause fails
- **WHEN** the pause for 3 A is refused by Easee
- **THEN** the pause-charging command is sent once and the caller is answered with one error report

#### Scenario: the pause is sent but the session report fails
- **WHEN** the pause for 0 A succeeds and the current-session report cannot be sent
- **THEN** the framework answers the caller with an error report

#### Scenario: a negative value
- **WHEN** the value is below zero
- **THEN** the framework answers with an error report and the controller is not called
