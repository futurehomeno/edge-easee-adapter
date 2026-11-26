## MODIFIED Requirements

### Requirement: Unauthorized Re-subscribe Reconnects Once
When a subscribe on an established connection, a re-sync or a charger's first subscribe, is refused
with the `HubException: Unauthorized` hub error, the manager SHALL restart the SignalR client once,
so it reconnects with the current access token. The connection carries the token it was opened with,
and nothing renews it while it lives. The restart SHALL be the client's own: it closes the connection
and launches a new one, and a `Close()` arriving before the relaunch, such as a logout or a manager
stop, SHALL win and leave the client closed. On a client that is not running or already closing the
restart SHALL do nothing, and SHALL NOT cancel a start that close deferred. The manager SHALL restart
at most once per connection, and at most once per charger until that charger next subscribes
successfully; a reconnect SHALL NOT reset that per-charger limit, so an Unauthorized that persists
after the reconnect, such as a charger that left the account, cannot restart the client again and
again. The manager SHALL NOT restart for any other error.

#### Scenario: re-sync refused as unauthorized
- **WHEN** a re-sync of a charger subscribed on the current connection is refused as unauthorized
- **THEN** the client is restarted once

#### Scenario: a logout or stop during the restart leaves the client closed
- **WHEN** the client is closed while the restart is closing the connection
- **THEN** the restart does not launch a new connection and the client stays closed

#### Scenario: restart of a client that is not running
- **WHEN** the restart reaches a client that is stopped, or closing with a start deferred
- **THEN** it starts nothing itself and the deferred start still runs

#### Scenario: second refusal on the same connection
- **WHEN** a second re-subscribe on the same connection is refused as unauthorized
- **THEN** the client is not restarted again

#### Scenario: refusal persists after the reconnect
- **WHEN** the subscribe of that charger on the new connection is refused as unauthorized
- **THEN** the client is not restarted

#### Scenario: first subscribe refused after connect
- **WHEN** the first subscribe of a charger on a connection is refused as unauthorized
- **THEN** the client is restarted once

#### Scenario: first subscribe refused again after the restart
- **WHEN** the subscribe of that charger on the new connection is refused as unauthorized too
- **THEN** the client is not restarted and the subscribe is retried with backoff

#### Scenario: first subscribe refused after the charger subscribed
- **WHEN** that charger has subscribed successfully since its refusal restarted the client, and a
  later first subscribe of it is refused as unauthorized
- **THEN** the client is restarted again

#### Scenario: other error
- **WHEN** a re-subscribe of a subscribed charger fails with another error
- **THEN** the client is not restarted
