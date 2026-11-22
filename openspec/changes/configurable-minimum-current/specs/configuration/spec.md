## MODIFIED Requirements

### Requirement: Settings And Defaults
The configuration SHALL expose these settings with the following defaults when unset or unparsable:
`pollingInterval` 20m, `token_refresh_interval` 30m, `currentWaitDuration` 3s,
`offered_current_wait_time` 30s, `energyLifetimeInterval` 10s, `httpTimeout` 30s,
`initial_charging_current` 16A, `min_charging_current` 6A, `use_locked_3_phases` false,
`auth_max_unauthorized` 2h, SignalR `connCreationTimeout` 30s, `keepAliveInterval2` 30s,
`timeoutInterval2` 1m and `invokeTimeout` 10s.
Duration settings are stored as strings and SHALL fall back to their default when they cannot be
parsed. The `min_charging_current` setter SHALL accept 1–32 A. The `initial_charging_current` setter
SHALL accept 6–32 A and never a value below `min_charging_current`. Setting `min_charging_current`
above the effective `initial_charging_current` (stored or default) SHALL raise the initial current to
the new minimum in the same update; lowering the minimum SHALL NOT change the initial current. A
rejected value SHALL fail with a range error and leave the stored value unchanged.

#### Scenario: unset duration
- **WHEN** a duration setting is absent from the stored configuration
- **THEN** its documented default is used

#### Scenario: unparsable duration
- **WHEN** a duration setting holds a string `time.ParseDuration` rejects
- **THEN** its documented default is used and no error is raised

#### Scenario: initial charging current not positive
- **WHEN** `initial_charging_current` is zero or negative
- **THEN** 16 is used

#### Scenario: three-phase lock unset
- **WHEN** `use_locked_3_phases` is absent from the stored configuration
- **THEN** it reads false

#### Scenario: minimum charging current not positive
- **WHEN** `min_charging_current` is zero or negative
- **THEN** 6 is used

#### Scenario: out-of-range minimum charging current
- **WHEN** 0 or 33 is written through the `min_charging_current` setter
- **THEN** the write is rejected with a range error and the stored value is left unchanged

#### Scenario: initial charging current below the minimum
- **WHEN** `min_charging_current` is 10 and an `initial_charging_current` of 8 is written
- **THEN** the write is rejected with a range error and the stored value is left unchanged

#### Scenario: the minimum is raised above the initial charging current
- **WHEN** `min_charging_current` is set to 20 while `initial_charging_current` is 16, stored or by default
- **THEN** both are stored as 20 in one update
- **AND** setting the minimum lower later leaves `initial_charging_current` at 20

### Requirement: Configuration Routes
The adapter SHALL expose get and set routes on the Easee service for `polling_interval`,
`current_wait_duration`, `easee_base_url`, `slow_charging_current_in_amperes`, `use_locked_3_phases`,
`initial_charging_current`, `min_charging_current`,
`http_timeout`, `signalr_base_url`, `signalr_conn_creation_timeout`, `signalr_keep_alive_interval`,
`signalr_timeout_interval`, `signalr_initial_backoff`, `signalr_repeated_backoff`,
`signalr_final_backoff`, `signalr_initial_failure_count`, `signalr_repeated_failure_count` and
`signalr_invoke_timeout`, alongside the framework's default route bundle.

#### Scenario: setting read over FIMP
- **WHEN** a config get command for one of these keys is received
- **THEN** the current value is reported

#### Scenario: setting written over FIMP
- **WHEN** a config set command for one of these keys is received
- **THEN** the value is stored and the change is published
