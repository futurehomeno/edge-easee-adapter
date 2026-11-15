## ADDED Requirements

### Requirement: Unauthorized Re-subscribe Reconnects Once
When a subscribe on an established connection is refused with the `HubException: Unauthorized` hub
error for a charger already subscribed on that same connection, the manager SHALL restart the
SignalR client once, so it reconnects with the current access token, through the same
close-then-start path Unregister uses. The connection carries the token it was opened with, and
nothing renews it while it lives. The manager SHALL restart at most once per connection, and SHALL
NOT restart for a charger not yet subscribed on the connection or for any other error, so an
Unauthorized that persists after the reconnect, such as a charger that left the account, cannot
restart the client again and again.

#### Scenario: re-sync refused as unauthorized
- **WHEN** a re-sync of a charger subscribed on the current connection is refused as unauthorized
- **THEN** the client is closed and started once

#### Scenario: second refusal on the same connection
- **WHEN** a second re-subscribe on the same connection is refused as unauthorized
- **THEN** the client is not restarted again

#### Scenario: refusal persists after the reconnect
- **WHEN** the subscribe of that charger on the new connection is refused as unauthorized
- **THEN** the client is not restarted

#### Scenario: first subscribe refused after connect
- **WHEN** the first subscribe of a charger on a connection is refused as unauthorized
- **THEN** the client is not restarted and the subscribe is retried with backoff

#### Scenario: other error
- **WHEN** a re-subscribe of a subscribed charger fails with another error
- **THEN** the client is not restarted
