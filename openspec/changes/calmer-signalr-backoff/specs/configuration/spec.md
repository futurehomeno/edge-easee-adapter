## MODIFIED Requirements

### Requirement: Backoff Settings
Both the authenticator and the SignalR client SHALL be driven by a stateful backoff configured from
initial, repeated and final durations plus an initial and repeated failure count. The authenticator
defaults to 1m / 5m / 10m and SignalR to 30s / 5m / 30m. A failure count of zero SHALL be read as 1.

#### Scenario: zero failure counts
- **WHEN** the initial or repeated failure count is zero
- **THEN** 1 is used in its place

#### Scenario: defaults applied
- **WHEN** no backoff durations are configured
- **THEN** the authenticator uses 1m / 5m / 10m and SignalR uses 30s / 5m / 30m

## ADDED Requirements

### Requirement: SignalR Backoff Schedule
The packaged SignalR backoff SHALL be 30 s for 3 failures, 5 min for the next 3, then 30 min, so
attempts after a reconnect run at 0, 30, 60, 90, 390, 690 and 990 s and then every 30 min. The
migration to config version 10 SHALL set all five stored SignalR backoff values (`initialBackoff`
`30s`, `repeatedBackoff` `5m`, `finalBackoff` `30m`, `initialFailureCount` 3, `repeatedFailureCount`
3) whatever they were, because an installed hub's stored values override the packaged defaults. A
value set on purpose is overwritten too.

#### Scenario: packaged schedule
- **WHEN** the packaged defaults drive the SignalR backoff
- **THEN** the delay is 30 s for failures 1-3, 5 min for failures 4-6 and 30 min from the seventh

#### Scenario: older stored schedule
- **WHEN** a version 9 configuration carries SignalR 5s / 30s / 10m with counts 2 / 3
- **THEN** it carries 30s / 5m / 30m with counts 3 / 3 and reaches version 10
