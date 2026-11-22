## ADDED Requirements

### Requirement: SignalR Repeated Failure Count
The packaged SignalR backoff SHALL be 10 s for 2 failures, 30 s for the next 8, then 10 min, so a
subscribe refused right after a reconnect is retried every 30 s until 260 s; Easee was seen to stop
refusing 110-135 s after the reconnect. The migration to config version 10 SHALL lift a stored SignalR
`repeatedFailureCount` below 8 to 8 and SHALL leave 8 or more alone, because an installed hub's stored
value overrides the packaged defaults.

#### Scenario: packaged schedule
- **WHEN** the packaged defaults drive the SignalR backoff
- **THEN** the delay is 30 s from the third through the tenth failure and 10 min from the eleventh

#### Scenario: old packaged count
- **WHEN** a version 9 configuration carries SignalR `repeatedFailureCount` 3
- **THEN** it becomes 8 and the configuration reaches version 10

#### Scenario: eight or more
- **WHEN** SignalR `repeatedFailureCount` is 8 or 12
- **THEN** it is left unchanged
