# Configuration Specification

## Purpose
Hold the adapter's tunable settings, expose them over FIMP config routes, migrate settings written by
older versions, and keep the Easee credentials in a store separate from the public configuration.

## Requirements

### Requirement: Settings And Defaults
The configuration SHALL expose these settings with the following defaults when unset or unparsable:
`pollingInterval` 10m, `token_refresh_interval` 30m, `currentWaitDuration` 3s,
`offered_current_wait_time` 20s, `energyLifetimeInterval` 10s, `httpTimeout` 30s,
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

### Requirement: Backoff Settings
Both the authenticator and the SignalR client SHALL be driven by a stateful backoff configured from
initial, repeated and final durations plus an initial and repeated failure count. The authenticator
defaults to 1m / 5m / 10m and SignalR to 5s / 30s / 10m. A failure count of zero SHALL be read as 1.

#### Scenario: zero failure counts
- **WHEN** the initial or repeated failure count is zero
- **THEN** 1 is used in its place

#### Scenario: defaults applied
- **WHEN** no backoff durations are configured
- **THEN** the authenticator uses 1m / 5m / 10m and SignalR uses 5s / 30s / 10m

### Requirement: Legacy Auth Backoff Migration
A configuration carrying the superseded `authenticatorBackoff` object SHALL have its five backoff
fields and its `maxUnauthorizedDuration` copied into `auth_backoff` and `auth_max_unauthorized`, and
the legacy field cleared. When the legacy object cannot be decoded the migration SHALL log a warning,
drop the legacy field and succeed, because returning an error would leave the config version
unchanged and the next startup would retry with the same broken bytes.

#### Scenario: legacy object present
- **WHEN** a configuration carrying `authenticatorBackoff` is migrated
- **THEN** its values are copied to the new keys and the legacy field is removed

#### Scenario: corrupt legacy object
- **WHEN** the legacy object is not valid JSON
- **THEN** a warning is logged, the field is dropped, and the migration reports success

#### Scenario: no legacy object
- **WHEN** the configuration carries no `authenticatorBackoff`
- **THEN** the migration is a no-op

### Requirement: Superseded Default Migrations
A stored `offered_current_wait_time` of exactly `15s` SHALL be rewritten to `20s`, and a stored
SignalR `finalBackoff` of exactly `2m` SHALL be rewritten to `10m`. Any other value SHALL be left
alone. A value deliberately chosen to match the old default is indistinguishable from it and is
rewritten too.

#### Scenario: old packaged default
- **WHEN** `offered_current_wait_time` is `15s`
- **THEN** it becomes `20s`

#### Scenario: user-chosen value
- **WHEN** `offered_current_wait_time` is `30s`
- **THEN** it is left unchanged

#### Scenario: SignalR final backoff
- **WHEN** the SignalR `finalBackoff` is `2m`
- **THEN** it becomes `10m`

### Requirement: Credential Migration From Config
A configuration at version 5 SHALL have its credentials moved out of `config.json` into the secrets
store and cleared from the configuration. A configuration carrying no credentials SHALL be left
alone. When the secrets store already holds credentials the config copy SHALL be dropped without
overwriting them, because an earlier run may have written the secrets and then failed to save the
version bump, and Easee retires a refresh token as soon as it is exchanged. A missing expiry SHALL
be backfilled from the token's `exp` claim; a claim that cannot be parsed SHALL leave the expiry
zero rather than fail the migration, because zero is the value the framework already treats as
"refresh now". A failed write SHALL be returned so the version does not advance and the next startup
retries with the tokens still in place.

#### Scenario: tokens still in the config
- **WHEN** the configuration carries credentials and the secrets store is empty
- **THEN** they are written to the secrets store and cleared from the configuration

#### Scenario: secrets already migrated
- **WHEN** the secrets store already holds credentials
- **THEN** the config copy is dropped and the stored credentials are kept

#### Scenario: token without a readable expiry claim
- **WHEN** a migrated token carries no parsable `exp` claim
- **THEN** its expiry is left zero and the migration succeeds

### Requirement: Configuration Routes
The adapter SHALL expose get and set routes on the Easee service for `polling_interval`,
`current_wait_duration`, `easee_base_url`, `slow_charging_current_in_amperes`, `http_timeout`,
`signalr_base_url`, `signalr_conn_creation_timeout`, `signalr_keep_alive_interval`,
`signalr_timeout_interval`, `signalr_initial_backoff`, `signalr_repeated_backoff`,
`signalr_final_backoff`, `signalr_initial_failure_count`, `signalr_repeated_failure_count` and
`signalr_invoke_timeout`, alongside the framework's default route bundle.

#### Scenario: setting read over FIMP
- **WHEN** a config get command for one of these keys is received
- **THEN** the current value is reported

#### Scenario: setting written over FIMP
- **WHEN** a config set command for one of these keys is received
- **THEN** the value is stored and the change is published

### Requirement: Configuration Report Redaction
`cmd.config.get_report` SHALL answer with the public configuration only, never the credentials. The
redaction SHALL return a copy, because aliasing the live model would race a concurrent update while
the report is marshalled.

#### Scenario: report requested
- **WHEN** the configuration report is requested
- **THEN** the public settings are returned and no token appears in the payload

### Requirement: Incoming Message Logging
Every incoming FIMP message carrying a payload SHALL be logged with its source, service, interface
and value; a message with a nil payload SHALL be skipped silently. The value
of any message whose interface begins with `cmd.auth.` SHALL be replaced with `***`. This routing
SHALL stay first in the routing table, because the router runs routings in order and the stats
callback fires only after handling.

#### Scenario: auth command received
- **WHEN** a message with an interface starting `cmd.auth.` is received
- **THEN** the log line carries `***` instead of the credentials

#### Scenario: ordinary command received
- **WHEN** any other message with a payload is received
- **THEN** its value is logged verbatim

#### Scenario: message without a payload
- **WHEN** a message with a nil payload is received
- **THEN** nothing is logged

### Requirement: Selection Store Locking
The `cmd.thing.delete` adapter route and the app configuration route SHALL share one message-handler
lock, so a thing deletion cannot interleave with a configuration write that rewrites the selection it
reads.

#### Scenario: concurrent delete and reconfigure
- **WHEN** a thing delete and an extended config set arrive together
- **THEN** they are serialised by the shared lock

### Requirement: Credential Store Separation
Credentials SHALL live in their own secrets store, not in the public configuration file, so that
resetting the configuration cannot leave the Easee tokens on the hub and the config report cannot
leak them.

#### Scenario: configuration reset
- **WHEN** the configuration is reset during uninstall
- **THEN** the credentials are cleared through their own store, reported as a separate error

### Requirement: Configuration Persisted At Initialization
Initialization SHALL save the configuration before evaluating the credentials, so that defaults and
completed migrations are written to disk even on a boot that ends unconfigured. A save failure SHALL
abort initialization.

#### Scenario: first boot
- **WHEN** the adapter initializes with no stored configuration
- **THEN** the defaults are written to disk before the credential check runs

#### Scenario: save fails
- **WHEN** persisting the configuration fails
- **THEN** initialization returns a `failed to save configs` error
