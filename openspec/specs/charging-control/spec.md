# Charging Control Specification

## Purpose
Translate FIMP chargepoint commands into Easee cloud calls: starting and stopping a session, setting
the offered (dynamic) current and the max current, and locking the cable. Easee has no "resume"
primitive the adapter can use, so a session is resumed by writing a dynamic current.
## Requirements
### Requirement: Connection Gate On Reads
Every report the controller produces SHALL first check that the charger is connected through the
SignalR manager, returning `charger <id> is not connected: <reason>` when it is not. Command setters
for max current and offered current SHALL NOT apply this gate.

#### Scenario: report requested while disconnected
- **WHEN** a state, session, cable-lock, phase-mode, meter or alarm report is requested for a charger
  the manager does not consider connected
- **THEN** the call fails with the disconnection reason

### Requirement: Start Charging
`cmd.charge.start` SHALL resume the charger by writing a dynamic current, because the Easee resume
command clears the dynamic current. The starting current SHALL be chosen as follows: the cached
requested offered current; if that is zero or less — meaning unknown, either because no load balancer
ever set one or because the session-finished observation cleared it — the cached max current
instead; otherwise, for a non-slow mode, raised to at least `initial_charging_current` (default 16A).
Slow mode SHALL be exempt from that floor and SHALL instead use `slowChargingCurrentInAmperes` when
that is greater than zero, rounded to the nearest ampere. A resulting current of zero SHALL fail with
`invalid start current`. A start Easee accepts but the charger never echoes back within the current
wait duration SHALL fail with `start accepted, but the charger did not resume at <current>A` - an
unconfirmed start may well have left the charger paused. A start the command channel defers SHALL
report success at once; the deferred send checks the echo.

#### Scenario: the charger never echoes the start back
- **WHEN** Easee accepts the start but no observation confirms the current within the wait duration
- **THEN** the command fails rather than reporting a start that may not have happened

#### Scenario: the start is deferred
- **WHEN** the start arrives within `offered_current_wait_time` of another command
- **THEN** the command succeeds without waiting for an echo, and the write goes out at the deadline

#### Scenario: normal start with a cached current below the floor
- **WHEN** the cached requested offered current is 10A and the initial charging current is 16A
- **THEN** the charger is started at 16A

#### Scenario: slow start
- **WHEN** the mode is `slow` and `slowChargingCurrentInAmperes` is 6
- **THEN** the charger is started at 6A, without the initial-charging-current floor

#### Scenario: slow start with no slow current configured
- **WHEN** the mode is `slow` and `slowChargingCurrentInAmperes` is 0
- **THEN** the cached requested offered current is used unchanged

#### Scenario: no cached current
- **WHEN** the cached requested offered current is zero or negative
- **THEN** the cached max current is used as the starting current

#### Scenario: nothing to start from
- **WHEN** both the cached requested offered current and the cached max current are zero
- **THEN** the command fails with `invalid start current`

### Requirement: Start Bypasses The Offered-Current Dedup
The start path SHALL bypass the recent-value dedup. If the user presses Start within
`offered_current_wait_time` of a Stop, the cached value still matches the start current — the cache
is cleared only asynchronously by the session-finished observation — and suppressing the call would
leave the charger stopped.

#### Scenario: start shortly after stop
- **WHEN** Start is issued within the offered-current wait time of a Stop, at the same current
- **THEN** the dynamic-current write is not suppressed: it is stored in the command channel, the
  command succeeds, and the charger resumes at the deadline

### Requirement: Stop Charging
`cmd.charge.stop` SHALL send the Easee pause-charging command through the per-charger command
channel, stamping the channel like any other send, because Easee sets the dynamic current to zero
on stop and ignores a dynamic-current write that follows it too closely.

#### Scenario: stop after a quiet period
- **WHEN** no command was sent for the charger within `offered_current_wait_time`
- **THEN** the pause-charging command is sent and a non-202 response is reported as an error

#### Scenario: stop too soon after a current change
- **WHEN** a dynamic-current write went out within `offered_current_wait_time` before the stop
- **THEN** the stop is stored rather than refused, and sent at the deadline

#### Scenario: write shortly after a stop
- **WHEN** a dynamic-current write arrives within `offered_current_wait_time` of a stop
- **THEN** it is stored and sent at the deadline the stop opened

### Requirement: Offered Current
`cmd.smart_charge.set` SHALL write the dynamic current through the per-charger command channel. The
requested value SHALL be clamped to the cached max current, or to 32A when no max current is cached,
logging the clamp. Unless forced, a value equal to the last requested value within
`offered_current_wait_time` SHALL be skipped. When the write goes out immediately, the requested
value SHALL be cached with the current hub time and the adapter SHALL wait up to
`current_wait_duration` (default 3s) for the charger to echo it back over SignalR; a deferred write
does both when it is sent. The wait SHALL resolve on either the value delivered by the notification
or the value the cache holds when it wakes: notifications are dropped on a full listener buffer, so
neither source alone sees every echo - a confirmation immediately superseded by a further
observation is gone from the cache, while a delivered value can itself be the stale one.

#### Scenario: the echo is immediately superseded
- **WHEN** the awaited current is observed and then at once replaced by a different value
- **THEN** the wait still succeeds, on the delivered notification

#### Scenario: value above the max
- **WHEN** 40A is requested and the cached max current is 32A
- **THEN** the clamp is logged and 32A is sent

#### Scenario: no max current cached
- **WHEN** 40A is requested and no max current is cached
- **THEN** the value is clamped to 32A

#### Scenario: repeated value inside the wait window
- **WHEN** the same current is requested again within `offered_current_wait_time`
- **THEN** no HTTP call is made and the command succeeds

#### Scenario: rate limit hit
- **WHEN** a different current is requested within `offered_current_wait_time` of the last send
- **THEN** the write is stored rather than refused, and the command succeeds

### Requirement: Max Current
`cmd.max_current.set` SHALL write the charger's max current and then wait up to
`current_wait_duration` for the charger to echo the new value over SignalR, so the forced report that
cliffhanger publishes afterwards carries the applied value.

#### Scenario: max current set
- **WHEN** a new max current is written successfully
- **THEN** the adapter waits up to `current_wait_duration` for the echoing observation and then
  returns successfully, whether or not the echo arrived

#### Scenario: write fails
- **WHEN** the Easee call fails
- **THEN** the error is returned and no wait happens

### Requirement: Supported Max Current Ceiling
The supported max current advertised for a charger SHALL be the site's rated current rounded to the
nearest ampere, capped at 32A.

#### Scenario: high rated current
- **WHEN** the site reports a rated current of 63A
- **THEN** the supported max current is 32A

### Requirement: Cable Lock Parameter
The parameters service SHALL expose exactly one parameter, `cable_always_locked`, as a boolean select
with `Yes`/`No` options defaulting to false. `cmd.param.set` SHALL send the lock-state command to
Easee and, on success, seed the cache with the written value, because the framework answers the
command with a forced report the reporting cache cannot suppress. The seed SHALL carry the zero
timestamp rather than the hub's current time, so that the observation stays authoritative and
overwrites it whenever it lands: observations are stamped with Easee's clock, and a hub running
ahead of the server would otherwise have the cache's timestamp guard reject the real value.
`cmd.param.get_report` SHALL answer from the cache. Any other parameter ID SHALL be rejected
with `parameter: <id> not supported`.

#### Scenario: lock the cable
- **WHEN** `cable_always_locked` is set to true
- **THEN** the Easee lock-state command is sent with state true

#### Scenario: the observation contradicts the seed
- **WHEN** an observation carries a timestamp behind the hub's clock after a seed was written
- **THEN** the observed value still replaces the seed

#### Scenario: unknown parameter
- **WHEN** any other parameter ID is set or read
- **THEN** the call fails with `parameter: <id> not supported`

#### Scenario: charger echoes the lock setting
- **WHEN** the lock-cable-permanently observation arrives
- **THEN** the cache is updated and a forced parameter report is published

### Requirement: Cable Lock Report
`evt.cable_lock.report` SHALL report the cached cable-lock flag. When the cable is unlocked the
reported cable current SHALL be 0. When it is locked, the cached cable current SHALL be included only
if it has been observed at least once and is non-negative; otherwise the report SHALL omit the
current entirely.

#### Scenario: cable unlocked
- **WHEN** the cached cable-lock flag is false
- **THEN** the report carries lock false and cable current 0

#### Scenario: cable locked with a known rating
- **WHEN** the cable is locked and a non-negative cable current has been observed
- **THEN** the report carries lock true and that current

#### Scenario: cable locked with no rating observed
- **WHEN** the cable is locked and no cable current has been observed
- **THEN** the report carries lock true and no cable current

### Requirement: Command Response Codes
The Easee command endpoints for dynamic current, max current, stop charging and cable lock SHALL be
treated as successful only on HTTP 202. The set-phase-mode endpoint SHALL accept both 200 and 202,
because Easee documents 200 but has been observed answering 202. Any other status SHALL be logged
with its body and returned as an error.

#### Scenario: unexpected status on a command
- **WHEN** a command endpoint answers a status outside its accepted set
- **THEN** the response body is logged and an error naming the command is returned

### Requirement: Charging Modes
The chargepoint service SHALL advertise exactly two charging modes, `normal` and `slow`. Mode
matching SHALL be case-insensitive.

#### Scenario: mixed-case mode
- **WHEN** a start command carries `Slow`
- **THEN** it is treated as slow mode

### Requirement: Deferred Offered-Current Writes
Every dynamic-current write and every pause for a charger SHALL go through one per-charger command
channel. The channel SHALL record when it last sent a command to Easee - a stop included - stamped
when the send is attempted, because a call that times out may still have reached Easee. A command
arriving while nothing is stored and at least `offered_current_wait_time` has passed since the last
send SHALL be sent at once. Any other command SHALL be stored in the channel's single slot and sent
at the last send time plus `offered_current_wait_time` (default 30s) - the moment the channel is
quiet again; a
newer command SHALL replace the stored one and SHALL NOT move that deadline, so a stream of commands
yields exactly one send per wait time, carrying the latest value. Storing a command SHALL log
`[<id>] Deferred <stop|set_current N>, sending in <d>` and the FIMP command SHALL succeed at once,
without waiting for the send. The deferred send SHALL run off the command goroutine and outside the
per-thing service lock; it SHALL stamp the channel, cache the requested current when the command is
a dynamic-current write, wait for the SignalR echo for logging only, and log a refused send at error
level. A stored command SHALL NOT survive an adapter restart.

#### Scenario: quiet channel
- **WHEN** a dynamic-current write arrives and at least `offered_current_wait_time` has passed since
  the last send
- **THEN** it is sent immediately

#### Scenario: write inside the window
- **WHEN** a dynamic-current write arrives within `offered_current_wait_time` of the last send
- **THEN** it is stored, the command succeeds, and the write is sent at the last send time plus
  `offered_current_wait_time`

#### Scenario: a stream of writes
- **WHEN** a new current arrives every 5s
- **THEN** Easee receives one write per wait time, carrying the latest value

#### Scenario: a start replaces a stored stop
- **WHEN** a stop is stored and a start arrives before the deadline
- **THEN** only the start is sent at the deadline and the charger is never paused

#### Scenario: the deferred send is refused
- **WHEN** Easee refuses the write when the deadline fires
- **THEN** the refusal is logged at error level; the command had already reported success

