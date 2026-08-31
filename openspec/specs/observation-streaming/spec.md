# Observation Streaming Specification

## Purpose
Maintain a single SignalR connection to the Easee cloud, keep every managed charger subscribed to it,
and dispatch the resulting observations into the per-charger cache and FIMP reports. This is the only
source of live charger telemetry; the HTTP API is used for commands and static configuration.

## Requirements

### Requirement: Single Shared Connection
One SignalR client SHALL serve every charger on the hub. The manager SHALL start the client when the
first charger registers and SHALL close it when the last charger unregisters. Starting an
already-running client SHALL be a no-op, and a start that races a close SHALL be dropped rather than
joining the wait group that close is already draining.

#### Scenario: first charger registers
- **WHEN** a charger is registered and the client is not connected
- **THEN** the client is started once, guarded so concurrent registrations do not start it twice

#### Scenario: start races a close
- **WHEN** the client is started while a close is still draining its connection goroutine
- **THEN** the start is dropped and no new connection goroutine is spawned

#### Scenario: last charger unregisters
- **WHEN** the final registered charger is unregistered
- **THEN** the charger is unsubscribed and the client is closed

#### Scenario: duplicate registration
- **WHEN** a charger ID that is already registered is registered again
- **THEN** a warning is logged and the existing registration is kept

### Requirement: Connection Retry With Backoff
The client SHALL reconnect on its own after a dropped or failed connection, spacing attempts with the
configured stateful backoff (`signalr` initial 5s, repeated 30s, final 10m by default). Two failure
modes add a fixed sleep before the backoff is consulted: when the access token cannot be obtained the
client SHALL sleep 1 minute, throttling the adapter's own reconnect loop; when the SignalR HTTP
connection cannot be established it SHALL sleep 30 seconds, throttling the underlying library's tight
retry loop. A successful connection SHALL reset the backoff.

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

### Requirement: Per-Charger Subscription
Each registered charger SHALL be subscribed on the shared connection before its observations are
delivered. Registration SHALL enqueue a subscription request on a buffered channel, handled by the
single manager run loop, only when the client already reports connected; while a connect is still in
flight the charger SHALL be left to the reconnect sweep, which already holds it, rather than enqueued
into a subscribe guaranteed to fail because the client is not running yet. A failed subscribe SHALL
arm a backoff-spaced retry rather than failing the registration.

#### Scenario: registered while the client is still connecting
- **WHEN** a charger is registered before the client reports connected
- **THEN** nothing is enqueued and the charger is subscribed by the `ClientStateConnected` sweep

#### Scenario: registered against a live connection
- **WHEN** a charger is registered while the client already reports connected
- **THEN** the subscription is enqueued straight away, there being no future connect to sweep it in

#### Scenario: subscribe succeeds
- **WHEN** the manager handles a queued subscription and the invoke succeeds
- **THEN** the charger is marked subscribed and its backoff is reset

#### Scenario: subscribe fails
- **WHEN** the subscribe invoke fails
- **THEN** a retry is armed after the charger's next backoff interval and the registration stands

#### Scenario: repeated failures stay quiet
- **WHEN** a charger keeps failing to subscribe
- **THEN** only the first failure of a streak logs at warning level and the rest log at debug level

#### Scenario: already subscribed
- **WHEN** a queued subscription is handled for a charger already marked subscribed
- **THEN** nothing is invoked, so a retry armed before a reconnect cannot subscribe twice

### Requirement: Resubscription On Reconnect
On a `ClientStateConnected` transition the manager SHALL clear the subscribed flag for every charger
and re-enqueue all of them, making the reconnect sweep the authority rather than trusting a
disconnect notice that may have been missed. It SHALL also clear each charger's failure streak so the
first failure on the new connection warns again. The re-enqueue SHALL happen off the run loop, which
is itself the only drainer of the subscription channel.

#### Scenario: client reconnects
- **WHEN** the client reports connected
- **THEN** every registered charger is marked unsubscribed and re-enqueued for subscription

#### Scenario: client disconnects
- **WHEN** the client reports disconnected
- **THEN** every charger's backoff is reset and its subscribed flag cleared

#### Scenario: unknown client state
- **WHEN** a client state other than connected or disconnected arrives
- **THEN** it is logged as a warning and otherwise ignored

### Requirement: Connectivity Reporting
`Connected(chargerID)` SHALL report false with a reason of `charger is not registered in a manager`,
`charger is not subscribed for SignalR observations` or `charger is offline`, in that order of
precedence. A charger counts as offline when either its cloud-connected flag or its charger-state
flag is false.

#### Scenario: unregistered charger
- **WHEN** connectivity is queried for a charger the manager does not hold
- **THEN** false is returned with `charger is not registered in a manager`

#### Scenario: registered but not subscribed
- **WHEN** the charger is registered and its subscribed flag is clear
- **THEN** false is returned with `charger is not subscribed for SignalR observations`

#### Scenario: charger reported offline
- **WHEN** the charger is subscribed but its handler reports not online
- **THEN** false is returned with `charger is offline`

### Requirement: Observation Dispatch
Observations SHALL be delivered to a buffered channel and dispatched serially by the single manager
run loop. An observation whose ID is not in the supported set SHALL be dropped silently. An
observation for an unregistered charger SHALL be reported as `no handler`. A handler error SHALL be
logged as a warning naming the charger and the observation.

#### Scenario: unsupported observation
- **WHEN** an observation arrives whose ID is not in `SupportedObservationIDs`
- **THEN** it is dropped without a log line

#### Scenario: observation for an unknown charger
- **WHEN** a supported observation names a charger that is not registered
- **THEN** the manager reports `no handler` and the run loop logs a warning

#### Scenario: handler returns an error
- **WHEN** the observation handler fails
- **THEN** a warning naming the charger ID and observation name is logged and dispatch continues

### Requirement: Supported Observations
The adapter SHALL handle these observation IDs: detected power grid type, phase mode, max charger
current, dynamic charger current, charger operating state, output phase, total power, lifetime
energy, energy session, cable rating, error code, in-current T3/T4/T5, cloud connected, cable locked,
lock cable permanently, charging session start and charging session stop.

#### Scenario: dispatch table coverage
- **WHEN** an observation with any of these IDs arrives for a registered charger
- **THEN** its dedicated handler runs

### Requirement: Timestamp-Guarded Cache Writes
Every cached value SHALL carry the timestamp of the observation that produced it, except
controller-requested values, which carry the time the command was issued. A write whose timestamp is
older than the value already held SHALL be rejected, and the handler SHALL return without publishing
a report. The session-finished clear of `requestedOfferedCurrent` follows the controller convention
rather than the observation one: it is written with the current time, not the timestamp of the state
observation that triggered it.

#### Scenario: session-finished clear carries the current time
- **WHEN** a session-finished state observation clears the cached requested offered current
- **THEN** the write carries the current time, so a state observation whose own timestamp is older
  than the last command still clears the value

#### Scenario: out-of-order observation
- **WHEN** an observation older than the cached value for the same key is handled
- **THEN** the cache write is rejected and no FIMP report is published

#### Scenario: newer observation
- **WHEN** the observation is newer than the cached value
- **THEN** the cache is updated and the associated report is published

### Requirement: Deduplicated Trace Logging
The handler SHALL trace-log an observation only when its value differs from the previously seen value
for the same observation ID, so a charger repeating an unchanged value does not flood the log.

#### Scenario: repeated identical value
- **WHEN** the same observation ID arrives twice with the same value
- **THEN** only the first is trace-logged

### Requirement: Online Flags From Observations
The cloud-connected observation SHALL drive the cloud online flag and the charger operating state
observation SHALL drive the state online flag, a charger being considered online only when both are
set. A charger operating state of offline SHALL clear the state flag.

#### Scenario: charger reports offline state
- **WHEN** the operating-state observation reports offline
- **THEN** the state online flag is cleared and connectivity reports the charger offline

### Requirement: Observation Value Typing
Observation values SHALL be parsed according to the data type the payload declares. Integer-typed
accessors SHALL reject a payload whose declared data type is not integer, and float accessors SHALL
reject one whose type is not double. A rejected parse SHALL surface as a handler error, except for
lifetime-energy observations: those are enqueued unparsed and parsed asynchronously in a background
goroutine, where a type error causes the observation to be skipped silently without reaching the
handler.

#### Scenario: mismatched data type
- **WHEN** an observation declares a data type the handler's accessor does not accept
- **THEN** the handler returns an error and the manager logs a warning

#### Scenario: mismatched data type on lifetime energy
- **WHEN** a lifetime-energy observation declares a data type that is not double
- **THEN** the background goroutine skips it and no handler error surfaces
