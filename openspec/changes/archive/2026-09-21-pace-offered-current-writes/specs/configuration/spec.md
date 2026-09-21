# Configuration

## MODIFIED Requirements

### Requirement: Settings And Defaults
The configuration SHALL expose these settings with the following defaults when unset or unparsable:
`pollingInterval` 10m, `token_refresh_interval` 30m, `currentWaitDuration` 3s,
`offered_current_wait_time` 30s, `energyLifetimeInterval` 10s, `httpTimeout` 30s,
`initial_charging_current` 16A, `auth_max_unauthorized` 2h, SignalR `connCreationTimeout` 30s,
`keepAliveInterval2` 30s, `timeoutInterval2` 1m and `invokeTimeout` 10s. Duration settings are stored
as strings and SHALL fall back to their default when they cannot be parsed.

#### Scenario: unset duration
- **WHEN** a duration setting is absent from the stored configuration
- **THEN** its documented default is used

#### Scenario: unparsable duration
- **WHEN** a duration setting holds a string `time.ParseDuration` rejects
- **THEN** its documented default is used and no error is raised

#### Scenario: initial charging current not positive
- **WHEN** `initial_charging_current` is zero or negative
- **THEN** 16 is used

### Requirement: Superseded Default Migrations
A stored `offered_current_wait_time` of exactly `15s` (the 2.8 packaged default) or `20s` (the
3.1.2 packaged default) SHALL be rewritten to `30s`, and a stored SignalR `finalBackoff` of exactly
`2m` SHALL be rewritten to `10m`. Any other value SHALL be left alone. A value deliberately chosen to
match an old default is indistinguishable from it and is rewritten too. The wait-time migration
SHALL run again for installs already past it, so a 3.1.2 install is lifted on its next boot.

#### Scenario: old packaged default
- **WHEN** `offered_current_wait_time` is `15s` or `20s`
- **THEN** it becomes `30s`

#### Scenario: user-chosen value
- **WHEN** `offered_current_wait_time` is `45s`
- **THEN** it is left unchanged

#### Scenario: SignalR final backoff
- **WHEN** the SignalR `finalBackoff` is `2m`
- **THEN** it becomes `10m`
