## ADDED Requirements

### Requirement: SignalR Hub Endpoint
The SignalR client SHALL connect to `<signalr_base_url>/hubs/chargers` with the header
`Authorization: Bearer <access token>`, giving up on creating the connection after `signalr_conn_creation_timeout`,
and SHALL run it with the keep-alive interval `signalr_keep_alive_interval` and the server timeout
`signalr_timeout_interval`. A charger SHALL be subscribed with the hub method `SubscribeWithCurrentState(<id>, true)`,
so Easee sends its current state first, and unsubscribed with `Unsubscribe(<id>)`. A hub call SHALL fail with
`timeout` when no result arrives within `signalr_invoke_timeout`, and with `client is not running` when the client
is stopped or has no connection object. A call on a connection still being set up waits for it: it succeeds when the
connection comes up and the result arrives within `signalr_invoke_timeout`, fails with `client closed` and the
connection's error when the setup fails within that time, and fails with `timeout` when that time runs out first. A
connection that drops is not resumed: a call on it fails with `client closed` and the connection's error, and the
client dials a new one.

#### Scenario: charger subscribed
- **WHEN** a charger is subscribed on a live connection
- **THEN** `SubscribeWithCurrentState` is invoked with its ID and `true`

#### Scenario: hub call without an answer
- **WHEN** a hub call gets no result within `signalr_invoke_timeout`
- **THEN** it fails with `timeout`

#### Scenario: hub call while disconnected
- **WHEN** a hub call is made while the client is not running
- **THEN** it fails with `client is not running` and nothing is sent
