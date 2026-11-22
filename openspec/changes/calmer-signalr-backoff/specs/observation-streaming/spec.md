## MODIFIED Requirements

### Requirement: Connection Retry With Backoff
The client SHALL reconnect on its own after a dropped or failed connection, spacing attempts with the
configured stateful backoff (`signalr` initial 30s, repeated 5m, final 30m by default). Two failure
modes add a fixed sleep before the backoff is consulted: when the access token cannot be obtained the
client SHALL sleep 1 minute, throttling the adapter's own reconnect loop; when the SignalR HTTP
connection cannot be established it SHALL sleep 30 seconds, throttling the underlying library's tight
retry loop. A successful connection SHALL reset the backoff. A superseded connection SHALL be
stopped before the next attempt, since each one holds a cancellable child of the client context that
is released only by stopping it.

#### Scenario: connection attempt fails
- **WHEN** establishing the SignalR connection fails
- **THEN** the client waits the next backoff interval and retries

#### Scenario: token provider fails
- **WHEN** the access token cannot be obtained
- **THEN** the client sleeps 1 minute before the backoff interval is added, and retries

#### Scenario: HTTP connection fails
- **WHEN** establishing the SignalR HTTP connection fails
- **THEN** the client sleeps 30 seconds before the backoff interval is added, and retries

#### Scenario: connection established
- **WHEN** the connection is established
- **THEN** the backoff is reset so the next outage starts from the initial interval
