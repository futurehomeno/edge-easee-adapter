# Spec coverage: requirements for behaviour the code has and no spec states

## Why
The OpenSpec cleanup of 2026-11-25 (step 4 of the repair plan, row E4) found that the `packaging` spec covers
only the service user's group and the re-owning of the data tree, while the Debian package also migrates state from
`/opt/thingsplex`, defends against planted symlinks, ships a systemd unit and four maintainer scripts. A sweep of
every package under `src/` and every FIMP route the adapter registers found a few more behaviours that are seen from
outside (files on disk, FIMP topics, Easee requests, start-up failures) and that no requirement states.

Every requirement below was written from the code on `release_3.4` (cliffhanger v1.5.5) and re-read against it;
every number was checked against its constant. The code is the reference: no code and no test change.

## What Changes
- `packaging`: ADDED Package Contents, Service Unit, Service Start And Stop In Maintainer Scripts, Purge Removes All
  State, Directory Layout At Configure, Symlink Defences In postinst, Legacy State Migration Trigger, Legacy State
  Migration, Legacy State Protection, Packaged Default Configuration.
- `configuration`: ADDED Secret File Modes Pinned At Startup, Config Backup Cleared At Startup, Unreadable Secrets
  File Starts Logged Out, Startup Stops On A Configuration Error, Log Defaults Reset On Migration To Version 2.
- `cloud-authentication`: ADDED Login Request, Application Login Lifecycle.
- `charging-control`: ADDED Easee HTTP Requests.
- `observation-streaming`: ADDED SignalR Hub Endpoint.
- `charger-discovery`: ADDED FIMP Topics And Route Bundles.

## New requirements: code and tests

| Requirement | Code (file, function) | Test |
|---|---|---|
| Package Contents | `Makefile` `configure`, `package-deb` | no test |
| Service Unit | `package/debian/usr/lib/systemd/system/easee.service` | no test |
| Service Start And Stop In Maintainer Scripts | `DEBIAN/preinst`; `DEBIAN/postinst` `configure_service` and the argument `case`; `DEBIAN/prerm`; `DEBIAN/postrm` | no test |
| Purge Removes All State | `DEBIAN/postrm` | no test |
| Directory Layout At Configure | `DEBIAN/postinst` `create_dirs`, `set_ownership`, `create_data_dir`, `set_data_modes` | no test (the group part of `set_data_modes` is the existing "Service User Group": `TestPostinst_DataModesGiveMigratedFilesFuturehomeGroup`) |
| Symlink Defences In postinst | `DEBIAN/postinst` `create_dirs`, `set_ownership`, `create_data_dir`, `set_data_modes` | no test |
| Legacy State Migration Trigger | `DEBIAN/postinst` `migrate_state` | no test |
| Legacy State Migration | `usr/lib/futurehome/easee/migrate.sh` | `TestMigrateScript_copiesLegacyState`, `TestMigrateScript_neverOverwritesExistingState`, `TestMigrateScript_sessionsWithoutDataDir` (`src/cmd/migrate_script_internal_test.go`); sync and failure on a broken copy: no test |
| Legacy State Protection | `usr/lib/futurehome/easee/migrate.sh` | `TestMigrateScript_closesLegacyTreeToOthers`, `TestMigrateScript_failsOnSymlinkedLegacyDataDir`, `TestMigrateScript_failsOnSymlinkToAnUntraversableTarget`, `TestMigrateScript_skipsSymlinkedSessionsAndLog`, modes in `TestMigrateScript_copiesLegacyState`; refusal as root: no test (the tests skip as root) |
| Packaged Default Configuration | `usr/share/futurehome/easee/defaults/config.json`; `cmd/factory.go` `migrateConfig` | `TestUninstall_resetAndReMigrate`, `TestPackagedDefaults_signalRBackoffSchedule`, `TestTelemetry_freshInstallIsEnabled`, `TestService_*_MatchesPackagedDefault`; MQTT, path and log values: no test |
| Secret File Modes Pinned At Startup | `config/credentials.go` `PinSecretModes`, called from `cmd/factory.go` `getCredentialsStore` | `TestPinSecretModes_ClosesTokenBearingFilesToOthers` |
| Config Backup Cleared At Startup | `config/credentials.go` `DropConfigBackup`, called from `cmd/factory.go` `migrateConfig` | `TestDropConfigBackup_RemovesTokenBearingBackup`, `TestMigrateConfig_dropsBackupLeftByAnInterruptedMigration`, `TestMigrateConfig_restoresTheBackupOverACorruptConfig` |
| Unreadable Secrets File Starts Logged Out | `cmd/factory.go` `getCredentialsStore`; `config/credentials.go` `DiscardLoaded` | `TestDiscardLoadedEmptiesAPartiallyDecodedModel` (the model only); the boot path, the log text and the start at version 5 with credentials: no test |
| Startup Stops On A Configuration Error | `cmd/factory.go` `getConfigService` (`log.Fatalf`) | `TestMigrateConfig_failedCredentialsMigrationSurfaces` (the error is returned); the exit: no test |
| Log Defaults Reset On Migration To Version 2 | `cmd/factory.go` `migrateConfig` (`resetLogDefaults`) | no test (`TestUninstall_resetAndReMigrate` sees `info` / `budzik`, but the packaged file already carries them) |
| Login Request | `api/authenticator.go` `Login`; `api/http.go` `httpClient.Login` | `TestLoginRejectsEmptyCredentialsWithoutCallingTheAPI`, `TestLoginFailureByStatusCode`, `TestClient_Login`; whitespace removal in the request: no test |
| Application Login Lifecycle | `app/app.go` `application.Login` | `TestApplication_Login` |
| Easee HTTP Requests | `api/http.go` `postCommand`, `getResponse`, `Ping` and the command and read methods | `TestClient_UpdateDynamicCurrent`, `TestClient_UpdateMaxCurrent`, `TestClient_StopCharging`, `TestClient_SetCableAlwaysLocked`, `TestClient_SetPhaseMode`, `TestClient_ChargerConfig`, `TestClient_Chargers`, `TestClient_Ping`; the site and details reads and the missing content type without a body: no test |
| SignalR Hub Endpoint | `signalr/client.go` `getConnection`, `getClient`, `SubscribeCharger`, `UnsubscribeCharger`, `invoke` | `TestEaseeAdapter` (`src/cmd/easee_internal_test.go`, against the test hub at `/hubs/chargers` in `src/internal/test/signalr.go`); the invoke timeout and "client is not running": no test |
| FIMP Topics And Route Bundles | `cmd/root.go` `Build`; `cmd/factory.go` `getMQTT`; `routing/router.go` `New` | `TestEaseeAdapter` (topics); service discovery and the route bundles: no test |

## Sweep: package and route to requirement

| Package / route | Covered by |
|---|---|
| `src` (`main.go`) | packaging (Service Unit: a failed run exits with status 1 and the unit restarts it); the goroutine dump and the error logged before the exit are log lines, not specified |
| `src/cmd` | configuration (migrations, Configuration Persisted At Initialization, the five new ones), packaging (Packaged Default Configuration), charger-discovery (FIMP Topics And Route Bundles), app-manifest (Routed Interfaces Declared, `TestPackagedManifest_DeclaresRoutedLogAndNodeInterfaces`) |
| `src/internal/api` | cloud-authentication (all; new Login Request), charging-control (Command Response Codes; new Easee HTTP Requests) |
| `src/internal/app` | charger-discovery (selection, manifest selector, re-seed, uninstall, charger list cache), cloud-authentication (Explicit Logout, Connection State From Ping; new Application Login Lifecycle), reporting (Diagnostic Error Report) |
| `src/internal/cache` | observation-streaming (Timestamp-Guarded Cache Writes) |
| `src/internal/config` | configuration (all), cloud-authentication (Credential Persistence, Stored Password Encrypted) |
| `src/internal/db` | reporting (Charging Session Storage, Session History Retrieval), charging-control (A User Stop Holds Until A Start) |
| `src/internal/easee` | charging-control, phase-mode, reporting, charger-discovery (Thing Composition, Thing State Refresh At Creation) |
| `src/internal/model` | phase-mode (Phase Mode Matrix, FIMP To Easee Mode Mapping), observation-streaming (Supported Observations, Observation Value Typing) |
| `src/internal/routing` | configuration (Configuration Routes, Selection Store Locking, Incoming Message Logging), charging-control (Sub-Minimum Current Pauses) |
| `src/internal/signalr` | observation-streaming (all; new SignalR Hub Endpoint), charging-control (Charging Watchdog), reporting (Connectivity Reported On Change) |
| `src/internal/tasks` | cloud-authentication (Periodic Token Refresh), reporting (Periodic Reporting Task) |
| `package/` | packaging (Service User Group; the ten new ones) |
| `cmd.config.get_report` | configuration (Configuration Report Redaction) |
| `cmd.config.get_<key>` / `set_<key>` for the 18 settings | configuration (Configuration Routes, Settings And Defaults) |
| `cmd.log.get_level` / `set_level` | app-manifest (Routed Interfaces Declared); behaviour is the framework's |
| `cmd.config.get_/set_` telemetry `enabled`, `validity`, `suppressed` | configuration (Telemetry Enabled By Default); behaviour is the framework's |
| `cmd.current_session.set_current` below 6 A | charging-control (Sub-Minimum Current Pauses) |
| `cmd.auth.login` | cloud-authentication (Login Request, Application Login Lifecycle, Password Login), charger-discovery (selection on login) |
| `cmd.auth.logout` | cloud-authentication (Explicit Logout) |
| `cmd.app.uninstall` | charger-discovery (Uninstall) |
| `cmd.config.extended_set` | charger-discovery (Configure Validates Before Mutating), configuration (Selection Store Locking) |
| `cmd.app.get_manifest` | charger-discovery (Manifest Device Selector, Manifest Charger List Cache) |
| `cmd.app.get_diag` | reporting (Diagnostic Error Report) |
| `cmd.thing.delete` | configuration (Selection Store Locking), charger-discovery (Charger Selection Model) |
| `cmd.thing.get_inclusion_report` | charger-discovery (Thing Composition) |
| `cmd.network.get_node` | app-manifest (Routed Interfaces Declared), reporting (Connectivity And Ping) |
| `cmd.app.get_state`, `cmd.config.get_extended_report`, `cmd.network.reset`, `cmd.network.get_all_nodes`, `cmd.ping.send` | charger-discovery (FIMP Topics And Route Bundles: registered); behaviour is the framework's |
| car-charger thing routes (chargepoint, meter_elec, parameters, alarm_system) | charging-control, reporting, phase-mode |

## Not specified, internal
- `src/internal/test/`: test helpers, fakes and the generated mocks.
- `src/cmd/factory.go`: the service container (lazy singletons, the authenticator reaching the application through
  a function), logger and telemetry set-up (a failed telemetry set-up is logged and telemetry stays off).
- `src/internal/api/client.go`: fetching the access token before each HTTP call and the debug log lines.
- `src/internal/signalr/logger.go`: maps the SignalR library's log lines onto logrus levels.
- `src/internal/app/app.go` `Check` and `CheckInterval`: no-ops the framework requires.
- `src/internal/api/http.go` `chargerSessionsURITemplate`: an unused constant, no behaviour.
- Log lines in general, except where an existing requirement names one.

## Not in this change
- The manifest's interface list: issue 48 waits for a decision.
- Behaviour that looks wrong or accidental is recorded as an issue, not as a requirement:
  - issue 49: the packaged config ships the legacy `authenticatorBackoff` object; the counts 4 / 8 come only from
    the migration 2→3 converting it;
  - issue 50: the unit's start limit (5 in 10 s) cannot trip with a 10 s restart delay;
  - issue 51: `polling_interval`, `http_timeout`, `easee_base_url` and the SignalR reconnect backoff set over FIMP
    take effect only after a restart;
  - issue 52: a failed login marks a running app not configured while its stored session stays.
- No existing requirement was found wrong against the code, so nothing is MODIFIED.
- A comment in `migrate.sh` says postinst's permission pass runs before the script, so the modes it sets stick;
  since `set_data_modes` was split out, postinst widens the migrated directories to 0755 after the script. The
  comment is stale; the behaviour is the one "Directory Layout At Configure" states.

## Impact
- Affected specs: `packaging`, `configuration`, `cloud-authentication`, `charging-control`, `observation-streaming`,
  `charger-discovery`
- Affected code: none
