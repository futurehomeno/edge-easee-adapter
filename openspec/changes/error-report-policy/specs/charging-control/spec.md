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

The framework checks an offered current only against 0 and `sup_max_current`; it does not reject a
current under 6 A. A value from 0 to 5 A SHALL still be handed to the controller in routing before
the framework's handler (issue 56): on success the message
interface SHALL be renamed with a `.paused` suffix so the framework's handler skips it, and the
adapter SHALL send the current-session report itself. A value from 6 A up to the minimum reaches the
controller through the framework and pauses there. The framework checks a max current against 6 A.

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
- **THEN** 3 A is written as the dynamic current and the message interface is renamed with a `.paused` suffix

#### Scenario: the pause fails
- **WHEN** handing a value under 6 A to the controller returns an error
- **THEN** the message is left unrenamed and the framework handles it normally

#### Scenario: the pause is sent but the session report fails
- **WHEN** the pause for 0 A succeeds and the current-session report cannot be sent
- **THEN** the caller is answered with an error report

#### Scenario: a negative value
- **WHEN** the value is below zero
- **THEN** the message is left untouched

## ADDED Requirements

### Requirement: A Refused Deferred Write Answers Its Command
A start, stop, offered current or phase mode command whose write the per-charger command channel
deferred is answered as accepted when it is stored. When Easee later refuses that write, the adapter
SHALL publish one `evt.error.report` for that command: its `corid` SHALL be the command's `uid`, its
value the failure's error code, and it SHALL be sent to the command's `resp_to` topic when set,
otherwise to the command's service on `mt:evt`. The failure SHALL be logged once, by that report.
A write displaced from the channel by a newer command is not reported.

#### Scenario: a deferred offered current is refused
- **WHEN** `cmd.current_session.set_current` with 12 is deferred and Easee answers the deferred write with 400
- **THEN** an `evt.error.report` with value `device_refused` and the command's `uid` as `corid` is published

#### Scenario: a displaced write
- **WHEN** a deferred write is replaced in the channel by a newer command before it is sent
- **THEN** no report answers the displaced command

### Requirement: Coded Command Failures
A command failing on the Easee cloud SHALL be answered with an error code and a plain sentence that
carries no URL, charger serial, token or response body; those SHALL reach the log only. The code
SHALL be `unauthorized` for HTTP 401 or 403, `not_found` for 404, `unavailable` for 429, a 5xx
status or a request that never got an answer, and `device_refused` for any other refusal. A charger
whose stream is not connected SHALL answer a report request with `unavailable`. The HTTP client
SHALL NOT log a failure it returns: the router logs it once.

#### Scenario: Easee is unreachable
- **WHEN** `cmd.charge.stop` fails because the Easee cloud does not answer
- **THEN** the error report carries `unavailable` and a sentence without a URL

#### Scenario: Easee refuses the credentials
- **WHEN** a command is answered 401 by Easee
- **THEN** the error report carries `unauthorized` and no log line is written by the HTTP client

#### Scenario: the stream is lost
- **WHEN** `cmd.state.get_report` arrives while the charger's stream is not connected
- **THEN** the error report carries `unavailable`
