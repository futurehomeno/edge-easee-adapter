# Observation Streaming

## ADDED Requirements

### Requirement: SignalR Waits For Login Without Warning
While no Easee session exists, the SignalR client SHALL treat a failed start as waiting for login:
it SHALL log that once at info per streak (debug afterwards), never as a warning, and SHALL start a
connection attempt as soon as `Start()` is called again (the login path), without waiting out its
backoff.

#### Scenario: chargers registered while logged out
- **WHEN** the client is started and the token provider reports "not logged in" on every attempt
- **THEN** no warning is logged and the waiting state is logged at info only once

#### Scenario: login while the client waits
- **WHEN** the user logs in and the login path calls `Start()` on the waiting client
- **THEN** the client asks for a token again immediately, not after its backoff
