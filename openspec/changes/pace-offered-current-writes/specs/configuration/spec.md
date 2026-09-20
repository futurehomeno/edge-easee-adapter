# Configuration

## MODIFIED Requirements

### Requirement: Settings And Defaults
The configuration SHALL expose these settings with the following defaults when unset or unparsable:
`pollingInterval` 10m, `token_refresh_interval` 30m, `currentWaitDuration` 3s,
`offered_current_wait_time` 20s, `offered_current_deferral_time` 45s, `energyLifetimeInterval` 10s,
`httpTimeout` 30s, `initial_charging_current` 16A, `auth_max_unauthorized` 2h, SignalR
`connCreationTimeout` 30s, `keepAliveInterval2` 30s, `timeoutInterval2` 1m and `invokeTimeout` 10s.
Duration settings are stored as strings and SHALL fall back to their default when they cannot be
parsed.

#### Scenario: unset duration
- **WHEN** a duration setting is absent from the stored configuration
- **THEN** its documented default is used

#### Scenario: unparsable duration
- **WHEN** a duration setting holds a string `time.ParseDuration` rejects
- **THEN** its documented default is used and no error is raised

#### Scenario: initial charging current not positive
- **WHEN** `initial_charging_current` is zero or negative
- **THEN** 16 is used
