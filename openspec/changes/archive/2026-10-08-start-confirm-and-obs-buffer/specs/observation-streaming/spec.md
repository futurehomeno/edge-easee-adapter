## ADDED Requirements

### Requirement: Observation Buffer Holds A Reconnect Replay
The SignalR receiver SHALL drop an observation whose ID is not in `SupportedObservationIDs` before it
enters the dispatch buffer. The buffer SHALL hold 500 observations. Easee replays each charger's
full snapshot on every (re)connect, mostly unsupported IDs, while the run loop may be busy with
synced session writes. The edge-triggered session start/stop replay must not be dropped for lack of
a slot.

#### Scenario: unsupported observation
- **WHEN** an observation with an unsupported ID arrives
- **THEN** it takes no buffer slot and is dropped without a log line

#### Scenario: replay while the run loop is stalled
- **WHEN** a 20-charger account reconnects and every charger replays all its supported observations before the run loop drains any
- **THEN** none of them is dropped
