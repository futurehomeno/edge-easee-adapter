## ADDED Requirements

### Requirement: FIMP Topics And Route Bundles
The adapter SHALL subscribe to the command topics `pt:j1/mt:cmd/rt:ad/rn:easee/ad:1` and
`pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/+/+`, SHALL publish with the source `easee`, and SHALL register for service
discovery as resource `easee` of type `ad`, instance `1`, with its package name and version. Next to its own routes
it SHALL register the framework's adapter routes (`cmd.thing.get_inclusion_report`, `cmd.thing.delete`,
`cmd.network.reset`, `cmd.network.get_node`, `cmd.network.get_all_nodes`, `cmd.ping.send`), the app routes
(`cmd.app.get_state`, `cmd.config.get_extended_report`, `cmd.app.get_manifest`, `cmd.config.extended_set`,
`cmd.app.uninstall`, `cmd.app.get_diag`, `cmd.auth.login`, `cmd.auth.logout`) and the car-charger thing routes.
The app has no reset and no token authorization, so `cmd.app.reset` and `cmd.auth.set_tokens` SHALL NOT be routed.

#### Scenario: command to a charger
- **WHEN** a command is published on `pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1`
- **THEN** the adapter receives and handles it

#### Scenario: inclusion report requested
- **WHEN** `cmd.thing.get_inclusion_report` is published on `pt:j1/mt:cmd/rt:ad/rn:easee/ad:1`
- **THEN** the inclusion report is published on `pt:j1/mt:evt/rt:ad/rn:easee/ad:1`

#### Scenario: token authorization sent
- **WHEN** `cmd.auth.set_tokens` is published for the `easee` service
- **THEN** no route handles it
