## ADDED Requirements

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

When `data/secrets.json` cannot be loaded but its backup can, the storage decodes the backup over the fields the
primary already filled, without clearing them, and the adapter starts with that mix: the backup's credentials plus any
field only the primary set.

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
