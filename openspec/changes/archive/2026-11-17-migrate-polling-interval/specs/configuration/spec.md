## MODIFIED Requirements

### Requirement: Settings And Defaults
The configuration SHALL expose these settings with the following defaults when unset or unparsable:
`pollingInterval` 20m, `token_refresh_interval` 30m, `currentWaitDuration` 3s,
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

## ADDED Requirements

### Requirement: Sub-Minute Polling Interval Migration
The migration to config version 9 SHALL rewrite a stored `pollingInterval` shorter than one minute
to `20m`, the `30s` packaged default up to 2.5.10 included. Each poll of an idle charger fetches its
config and site info over REST. An interval of one minute or longer SHALL be left alone, and so SHALL
an unset or unparsable one, which the getter already reads as its `20m` fallback.

#### Scenario: 2.5 polling interval
- **WHEN** a version 8 configuration carries `pollingInterval` `30s`
- **THEN** it becomes `20m` and the configuration reaches version 9

#### Scenario: one minute or longer
- **WHEN** `pollingInterval` is `1m` or `5m`
- **THEN** it is left unchanged

#### Scenario: unset or unparsable
- **WHEN** `pollingInterval` is empty or `soon`
- **THEN** it is left unchanged
