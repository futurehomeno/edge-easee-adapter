## ADDED Requirements

### Requirement: A Refused Deferred Write Answers Its Command
A start, stop, offered current or phase mode command whose write the per-charger command channel
deferred is answered as accepted when it is stored. When Easee later refuses that write, the adapter
SHALL publish one `evt.error.report` for that command: its `corid` SHALL be the command's `uid`, its
value the failure's error code, and it SHALL be sent to the command's `resp_to` topic when set,
otherwise to the command's service on `mt:evt`. Its `msg` SHALL NOT name the charger serial. The
failure SHALL be logged once, by that report. A write displaced from the channel by a newer command
is not reported.

#### Scenario: a deferred offered current is refused
- **WHEN** `cmd.current_session.set_current` with 12 is deferred and Easee answers the deferred write with 400
- **THEN** an `evt.error.report` with value `device_refused` and the command's `uid` as `corid` is published

#### Scenario: a deferred write fails without a code
- **WHEN** a deferred write fails because no access token can be had
- **THEN** the `evt.error.report` carries `internal` and its `msg` does not name the charger serial

#### Scenario: a displaced write
- **WHEN** a deferred write is replaced in the channel by a newer command before it is sent
- **THEN** no report answers the displaced command

### Requirement: Only Commands Are Bound
The adapter SHALL bind each chargepoint command to its charger's controller from the first route to
the last, so commands of one charger run one at a time. A report request (a message type ending in
`.get_report`) SHALL NOT be bound, so it does not wait for the binding of a running command of the
same charger. The framework's chargepoint service still holds its own lock while a command's
controller call runs, and a report request waits for that call.

#### Scenario: report request during a bound command
- **WHEN** `cmd.state.get_report` arrives for a charger while a command for that charger is bound
- **THEN** the report request is not held by that binding

### Requirement: Coded Command Failures
A command failing on the Easee cloud SHALL be answered with an error code and a plain sentence that
carries no URL, charger serial, token or response body; those SHALL reach the log only. The code
SHALL be `unauthorized` for HTTP 401 or 403, `not_found` for 404, `rate_limited` for 429, `unavailable` for a 5xx
status or a request that never got an answer, and `device_refused` for any other refusal. The log
line SHALL carry the request's method and URL. A charger whose stream is not connected, or whose
phase mode cannot be mapped, SHALL answer a report request with `unavailable`. The HTTP client
SHALL NOT log a failure it returns: the router logs it once.

#### Scenario: Easee is unreachable
- **WHEN** `cmd.charge.stop` fails because the Easee cloud does not answer
- **THEN** the error report carries `unavailable` and a sentence without a URL

#### Scenario: Easee refuses the credentials
- **WHEN** a command is answered 401 by Easee
- **THEN** the error report carries `unauthorized` and no log line is written by the HTTP client

#### Scenario: Easee limits requests
- **WHEN** a command is answered 429 by Easee
- **THEN** the error report carries `rate_limited` and a sentence without a URL

#### Scenario: the stream is lost
- **WHEN** `cmd.state.get_report` arrives while the charger's stream is not connected
- **THEN** the error report carries `unavailable`

#### Scenario: the phase mode cannot be mapped
- **WHEN** `cmd.phase_mode.get_report` arrives for a charger whose grid type is not detected yet
- **THEN** the error report carries `unavailable` and a sentence without the charger serial
