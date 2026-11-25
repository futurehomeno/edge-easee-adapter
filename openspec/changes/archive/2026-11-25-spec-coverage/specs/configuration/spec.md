## ADDED Requirements

### Requirement: Secret File Modes Pinned At Startup
On every start the adapter SHALL set `data/secrets.json`, `data/secrets.json.bak` and `data/config.json.bak` to mode
0640, because the storage enforces that mode when it saves a file but not when it loads one, and postinst cannot
change it safely as root. A missing file SHALL be skipped; any other failure SHALL be logged as a warning and the
start SHALL continue.

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

#### Scenario: corrupt configuration with a good backup
- **WHEN** `data/config.json` does not decode and `data/config.json.bak` does
- **THEN** the backup is moved into place unchanged and the next start loads it

### Requirement: Unreadable Secrets File Starts Logged Out
When the secrets file cannot be loaded at start, the adapter SHALL start logged out: it SHALL empty the in-memory
credentials, because a file that fails to decode partway leaves the fields already read behind, SHALL keep the file
on disk, and SHALL log an error that omits the load error, because that error carries the file body with the
tokens. The start SHALL continue, so a corrupt file does not cause a restart loop and a new login rewrites it.

#### Scenario: corrupt secrets file
- **WHEN** `data/secrets.json` holds an access token followed by bytes that do not decode
- **THEN** the app starts not configured, the file is still on disk, and no log line contains the token

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
