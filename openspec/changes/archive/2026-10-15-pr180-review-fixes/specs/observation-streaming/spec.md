## MODIFIED Requirements

### Requirement: On-Demand State Re-sync
The manager SHALL accept a state re-sync request for a registered charger and, when the client is connected and the charger is subscribed, invoke `SubscribeWithCurrentState(id, true)` again, bypassing the already-subscribed skip, so Easee replays the charger's current observations. The replay SHALL go through the normal observation dispatch. A request for a charger that is not subscribed, or while the client is not connected, SHALL invoke nothing, because the reconnect sweep or the pending subscription already brings a snapshot. The invoke SHALL follow the same rules as a subscribe: off the manager lock, bounded by the invoke timeout, and discarded if the charger is unregistered or the connection is replaced meanwhile. A failed re-sync SHALL be logged, SHALL NOT mark the charger unsubscribed, and SHALL be retried with the subscribe backoff.

#### Scenario: subscribed charger
- **WHEN** a re-sync is requested for a subscribed charger on a connected client
- **THEN** `SubscribeWithCurrentState` is invoked for it and its replayed observations are dispatched

#### Scenario: charger not subscribed
- **WHEN** a re-sync is requested for a charger that is not subscribed
- **THEN** nothing is invoked

#### Scenario: client disconnected
- **WHEN** a re-sync is requested while the client is not connected
- **THEN** nothing is invoked

#### Scenario: re-sync fails
- **WHEN** the re-sync invoke fails or times out
- **THEN** the failure is logged, the charger stays subscribed and the re-sync is retried
