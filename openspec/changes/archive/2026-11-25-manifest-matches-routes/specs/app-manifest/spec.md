## MODIFIED Requirements

### Requirement: Routed Interfaces Declared
The packaged app manifest SHALL declare, on its one service (the adapter's app service), exactly the messages routed on that service: every command a route handles as `in` and every event those routes send as `out`, each with the value type the handler reads or sends. The events include `evt.error.report` (string), which the framework's message handler sends in reply to any routed command whose handler returns an error. A message that no route handles or sends SHALL NOT be declared. An optional command of cliffhanger's `app.RouteApp` (`cmd.auth.login`, `cmd.auth.logout`, `cmd.auth.set_tokens`, `cmd.app.reset`) SHALL be declared exactly when the app implements the matching capability interface. The per-setting routes `cmd.config.get_<setting>` and `cmd.config.set_<setting>` and their `evt.config.<setting>_report` events, the telemetry settings included, SHALL NOT be declared; the `configuration` capability lists those settings.

#### Scenario: manifest lists the pairs
- **WHEN** a client reads the packaged manifest
- **THEN** it finds `cmd.log.get_level` and `cmd.log.set_level` with `evt.log.level_report`, and `cmd.network.get_node` with `evt.network.node_report`, each with its direction

#### Scenario: a routed command and its reply are declared
- **WHEN** a client reads the packaged manifest
- **THEN** it finds each command routed on the app service as `in`, for example `cmd.app.get_state` with value type `null`, and each reply as `out`, for example `evt.app.state_report` with value type `object`

#### Scenario: the error report is declared
- **WHEN** a client reads the packaged manifest
- **THEN** it finds `evt.error.report` as `out` with value type `string`

#### Scenario: a message nothing routes is not declared
- **WHEN** a client reads the packaged manifest
- **THEN** it finds none of `cmd.app.get_full_state`, `cmd.app.full_state_report`, `cmd.thing.inclusion` and `evt.auth.login_report`

#### Scenario: an optional command follows the app's capabilities
- **WHEN** the app implements `Login` and `Logout` but not `Authorize`
- **THEN** the manifest declares `cmd.auth.login` and `cmd.auth.logout` with `evt.auth.status_report`, and does not declare `cmd.auth.set_tokens`

#### Scenario: per-setting routes are left out
- **WHEN** a client reads the packaged manifest
- **THEN** it finds `cmd.config.get_report` and `cmd.config.get_extended_report`, and no `cmd.config.get_<setting>` or `cmd.config.set_<setting>` route of a single setting
