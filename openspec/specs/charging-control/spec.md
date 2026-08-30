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
`invalid start current`.

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
The start path SHALL force the dynamic-current write through, bypassing the recent-value dedup. If
the user presses Start within `offered_current_wait_time` of a Stop, the cached value still matches
the start current — the cache is cleared only asynchronously by the session-finished observation — and
suppressing the call would leave the charger stopped.

#### Scenario: start shortly after stop
- **WHEN** Start is issued within the offered-current wait time of a Stop, at the same current
- **THEN** the dynamic-current write is still sent to Easee

### Requirement: Stop Charging
`cmd.charge.stop` SHALL call the Easee pause-charging command. The HTTP layer SHALL refuse it when a
dynamic-current write for the same charger landed within `offered_current_wait_time` (default 20s),
returning `too many requests to the charger`, because Easee sets the dynamic current to zero on stop.

#### Scenario: stop after a quiet period
- **WHEN** no recent dynamic-current write was made for the charger
- **THEN** the pause-charging command is sent and a non-202 response is reported as an error

#### Scenario: stop too soon after a current change
- **WHEN** a dynamic-current write landed within the wait time
- **THEN** the stop is refused locally with `too many requests to the charger`

### Requirement: Offered Current
`cmd.smart_charge.set` SHALL write the dynamic current. The requested value SHALL be clamped to the
cached max current, or to 32A when no max current is cached, logging the clamp. Unless forced, a
value equal to the last requested value within `offered_current_wait_time` SHALL be skipped. After a
successful write the requested value SHALL be cached with the current hub time, and the adapter SHALL
wait up to `current_wait_duration` (default 3s) for the charger to echo it back over SignalR.

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
- **WHEN** the HTTP layer refuses the write because another change landed within the wait time
- **THEN** the command fails with `too many requests`

### Requirement: Max Current
`cmd.max_current.set` SHALL write the charger's max current and then wait up to
`current_wait_duration` for the charger to echo the new value over SignalR, so the forced report that
cliffhanger publishes afterwards carries the applied value.

#### Scenario: max current set
- **WHEN** a new max current is written successfully
- **THEN** the adapter waits for the echoing observation before returning

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
Easee; `cmd.param.get_report` SHALL answer from the cache. Any other parameter ID SHALL be rejected
with `parameter: <id> not supported`.

#### Scenario: lock the cable
- **WHEN** `cable_always_locked` is set to true
- **THEN** the Easee lock-state command is sent with state true

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
