## MODIFIED Requirements

### Requirement: Charging Session Storage
Charging sessions SHALL be persisted per charger in a buntdb bucket named
`charging-sessions:<chargerID>`, keyed by the Easee session ID. Registering a session start SHALL
first close a previous session left open, by stamping its stop time with the new session's start
time. Registering a session start whose ID is already stored SHALL leave that session unchanged.
Registering a session stop SHALL write the session's start, stop and energy.

#### Scenario: previous session left open
- **WHEN** a session start is registered and the latest stored session has a zero stop time
- **THEN** that session's stop is set to the new session's start before the new one is stored

#### Scenario: start replayed after its stop
- **WHEN** a session start is registered for a session ID that is already stored with a stop time
- **THEN** the stored session keeps its start, stop and energy

#### Scenario: start of an older stored session replayed
- **WHEN** a session start is registered for a stored session ID older than the latest session
- **THEN** both sessions stay unchanged and the latest one is not closed

#### Scenario: session stop
- **WHEN** a session stop observation arrives
- **THEN** the session's start, stop and energy are written under its ID

#### Scenario: session observations publish a report
- **WHEN** either a session start or a session stop is registered successfully
- **THEN** an unforced current-session report is published
