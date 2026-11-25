# Configuration Specification

## Purpose
Hold the adapter's tunable settings, expose them over FIMP config routes, migrate settings written by
older versions, and keep the Easee credentials in a store separate from the public configuration.
## Requirements
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

#### Scenario: the minimum and the initial charging current are written at the same time
- **WHEN** `min_charging_current` 20 and `initial_charging_current` 18 are written concurrently
- **THEN** `initial_charging_current` is stored no lower than the minimum, whichever write lands first

### Requirement: Backoff Settings
Both the authenticator and the SignalR client SHALL be driven by a stateful backoff configured from
initial, repeated and final durations plus an initial and repeated failure count. The authenticator
defaults to 1m / 5m / 10m and SignalR to 60s / 5m / 30m. A failure count of zero SHALL be read as 1.

#### Scenario: zero failure counts
- **WHEN** the initial or repeated failure count is zero
- **THEN** 1 is used in its place

#### Scenario: defaults applied
- **WHEN** no backoff durations are configured
- **THEN** the authenticator uses 1m / 5m / 10m and SignalR uses 60s / 5m / 30m

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

### Requirement: Credential Migration From Config
A configuration at version 5 SHALL have its credentials moved out of `config.json` into the secrets
store and cleared from the configuration. A configuration carrying no credentials SHALL be left
alone. When the secrets store already holds credentials the config copy SHALL be dropped without
overwriting them, because an earlier run may have written the secrets and then failed to save the
version bump, and Easee retires a refresh token as soon as it is exchanged. A missing expiry SHALL
be backfilled from the token's `exp` claim; a claim that cannot be parsed SHALL leave the expiry
zero rather than fail the migration, because zero is the value the framework already treats as
"refresh now" for the access token and as "no local expiry check" for the refresh one, so the
session recovers on the first exchange instead of needing a login. A failed write SHALL be returned
so the version does not advance and the next startup retries with the tokens still in place.

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

### Requirement: Telemetry Enabled By Default
A configuration without a telemetry block SHALL get cliffhanger's default: enabled, with the 30-day
validity starting at startup. The migration to config version 8 SHALL switch a stored block that is
disabled with a zero `enabled_at` (the seed written by earlier versions) on, keeping its validity and
suppression, with its window starting at startup, and SHALL run only once.

#### Scenario: fresh install
- **WHEN** the adapter starts with no telemetry block
- **THEN** telemetry is enabled

#### Scenario: old disabled seed
- **WHEN** a version 7 configuration carries a disabled telemetry block with a zero `enabled_at`
- **THEN** the migration enables it and the configuration reaches version 8

#### Scenario: already migrated
- **WHEN** a version 8 configuration carries a disabled telemetry block
- **THEN** it is left disabled

### Requirement: SignalR Backoff Schedule
The packaged SignalR backoff SHALL be 60 s for 3 failures, 5 min for the next 3, then 30 min, so
attempts after a reconnect run at 0, 60, 120, 180, 480, 780 and 1080 s and then every 30 min. The
migration to config version 10 SHALL set all five stored SignalR backoff values (`initialBackoff`
`60s`, `repeatedBackoff` `5m`, `finalBackoff` `30m`, `initialFailureCount` 3, `repeatedFailureCount`
3) whatever they were, because an installed hub's stored values override the packaged defaults. A
value set on purpose is overwritten too.

#### Scenario: packaged schedule
- **WHEN** the packaged defaults drive the SignalR backoff
- **THEN** the delay is 60 s for failures 1-3, 5 min for failures 4-6 and 30 min from the seventh

#### Scenario: older stored schedule
- **WHEN** a version 9 configuration carries SignalR 5s / 30s / 10m with counts 2 / 3
- **THEN** it carries 60s / 5m / 30m with counts 3 / 3 and reaches version 10

### Requirement: Secret File Modes Pinned At Startup
On every start that loads the configuration the adapter SHALL set `data/secrets.json`, `data/secrets.json.bak` and
`data/config.json.bak` to mode 0640, because the storage enforces that mode when it saves a file but not when it
loads one, and postinst cannot change it safely as root. A missing file SHALL be skipped; any other failure SHALL be
logged as a warning and the start SHALL continue. A start that stops because the configuration does not load
("Startup Stops On A Configuration Error") exits before this step and leaves the modes as they were.

#### Scenario: world-readable secrets
- **WHEN** the adapter starts with the three files at mode 0644
- **THEN** all three are 0640

#### Scenario: files absent
- **WHEN** none of the three files exists
- **THEN** the start continues without a warning

### Requirement: Config Backup Cleared At Startup
After the configuration migrations, on every start, the adapter SHALL remove `data/config.json.bak` when
`data/config.json` loads, because the backup made by a save can still hold the tokens that the move to the secrets
store took out of the configuration. When `data/config.json` is missing or does not decode, the backup SHALL be
renamed into its place instead, so the next start has a configuration to load. A failure SHALL be logged as a
warning and SHALL NOT stop the start.

#### Scenario: backup left by an interrupted migration
- **WHEN** a version 6 configuration starts next to a `config.json.bak` holding an access token
- **THEN** the backup is removed

#### Scenario: corrupt configuration with a good backup at the latest version
- **WHEN** `data/config.json` does not decode and `data/config.json.bak` decodes at the latest configuration version
- **THEN** the backup is moved into place unchanged and the next start loads it

#### Scenario: corrupt configuration with an older backup
- **WHEN** `data/config.json` does not decode and `data/config.json.bak` decodes at an older configuration version
- **THEN** the migrations run on the backup, their save writes a new `data/config.json`, and the corrupt file it moved
  to the backup is removed

### Requirement: Unreadable Secrets File Starts Logged Out
When neither the secrets file nor its backup `data/secrets.json.bak` can be loaded at start, the adapter SHALL start
logged out: it SHALL empty the in-memory
credentials, because a file that fails to decode partway leaves the fields already read behind, SHALL keep the file
on disk, and SHALL log an error that omits the load error, because that error carries the file body with the
tokens. The start SHALL continue, so a corrupt file does not cause a restart loop and a new login rewrites it.

When `data/secrets.json` cannot be loaded but its backup can, the storage loads the backup and the adapter starts
with those credentials.

One start differs: when the configuration is still at version 5 and carries credentials, the migration to version 6
finds the emptied store and saves those credentials as a new secrets file, the unreadable one becoming its backup,
so the adapter starts logged in.

#### Scenario: corrupt secrets file
- **WHEN** `data/secrets.json` holds an access token followed by bytes that do not decode and the configuration carries
  no credentials
- **THEN** the app starts not configured, the file is still on disk, and no log line contains the token

#### Scenario: corrupt secrets file next to a version 5 configuration with credentials
- **WHEN** `data/secrets.json` does not decode and `data/config.json` is at version 5 with an access and a refresh
  token
- **THEN** the migration to version 6 writes those tokens to a new `data/secrets.json` and the app starts configured

### Requirement: Startup Stops On A Configuration Error
When the configuration cannot be loaded, or a configuration migration fails, the adapter SHALL log the error and
exit instead of starting with partial settings, so the next start, by systemd's restart, tries again.

#### Scenario: credential migration cannot write the secrets
- **WHEN** the migration from version 5 to 6 fails to write the secrets file
- **THEN** the migration error, prefixed `migrate config`, is logged and the adapter exits

### Requirement: Log Defaults Reset On Migration To Version 2
The migration of a configuration at version 0 or 1 to version 2 SHALL set `log_level` to `info` and `log_format` to
`budzik`, whatever they were.

#### Scenario: old configuration with debug logging
- **WHEN** a version 1 configuration carries `log_level` `debug`
- **THEN** after the migrations it carries `info` and `budzik`

