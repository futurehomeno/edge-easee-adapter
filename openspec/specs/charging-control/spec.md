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
instead; if that is also zero — meaning the cache has not been seeded by an observation since the
last restart — 32A, the same ceiling the offered-current write already falls back to, so a restart
does not leave the charger unstartable until observation 47 arrives; otherwise, for a non-slow mode,
raised to at least `initial_charging_current` (default 16A).
Slow mode SHALL be exempt from that floor and SHALL instead use `slowChargingCurrentInAmperes` when
that is greater than zero, rounded to the nearest ampere. A resulting current of zero SHALL fail with
`invalid start current`. A start Easee accepts but the charger never echoes back within the current
wait duration SHALL fail with `start accepted, but the charger did not resume at <current>A` - an
unconfirmed start may well have left the charger paused.

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
- **THEN** the charger is started at 32A rather than failing with `invalid start current`, because
  an unseeded cache after a restart is not the same as a charger that cannot charge

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
Easee, treating both `200` and `202` as success, and, on success, seed the cache with the written value, because the framework answers the
command with a forced report the reporting cache cannot suppress. The seed SHALL carry the zero
timestamp rather than the hub's current time, so that the observation stays authoritative and
overwrites it whenever it lands: observations are stamped with Easee's clock, and a hub running
ahead of the server would otherwise have the cache's timestamp guard reject the real value.
`cmd.param.get_report` SHALL answer from the cache. Any other parameter ID SHALL be rejected
with `parameter: <id> not supported`.

#### Scenario: lock the cable
- **WHEN** `cable_always_locked` is set to true
- **THEN** the Easee lock-state command is sent with state true

#### Scenario: Easee answers 200
- **WHEN** the lock-state command is answered with `200 OK`
- **THEN** the set succeeds

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
`[<id>] <stop|set_current N>, delayed <d>` and the FIMP command SHALL succeed at once,
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

#### Scenario: storing a command is logged
- **WHEN** a `set_current 26` is stored with 15s left in the window
- **THEN** `set_current 26, delayed 15s` is logged for that charger

### Requirement: Deferred Write Echo Wait
A deferred dynamic-current write SHALL wait up to `offered_current_wait_time` for the charger to echo
it, not `current_wait_duration`. A paused charger often echoes after `current_wait_duration`, and
nothing waits on a deferred send's answer. A missing echo SHALL still be logged as a warning once
that wait ends. Teardown SHALL wait for a deferred send that has started until it reaches Easee, but
not for its echo.

#### Scenario: late echo of a deferred write
- **WHEN** a deferred write is sent and the charger echoes it 6s later
- **THEN** no `was not echoed back` warning is logged

#### Scenario: no echo within the window
- **WHEN** a deferred write is sent and no echo arrives within `offered_current_wait_time`
- **THEN** `Deferred set_current N was not echoed back by the charger` is logged as a warning

#### Scenario: teardown during the echo wait
- **WHEN** the thing is deleted while a sent deferred write is still waiting for its echo
- **THEN** Teardown returns without waiting for the echo

### Requirement: The Last Command Always Wins
The command stored in the per-charger channel SHALL always be the most recent one, whatever kind it
is and whichever command issued it: a stop SHALL displace a stored dynamic-current write, and a
dynamic-current write - an explicit `cmd.charge.start` among them - SHALL displace a stored stop.
The displaced command SHALL NOT reach Easee, its own command having already reported success, and
the deadline SHALL NOT move. The dedup that suppresses a repeated current SHALL NOT apply while a
stop is stored, so a write carrying the last-sent value displaces that stop like any other.

Displacing a stored command SHALL be logged as `[<id>] <new> preempts <old>`, naming both, so the
log shows which command was dropped and what replaced it rather than only what was stored. When the
displaced command is a pair, both halves SHALL be named, or a discarded resume would leave no trace.

The charger acts on whatever it last heard, so the hub's latest intent is the only one worth
sending; holding an older command in the slot makes the adapter answer a command it did not carry
out. This supersedes the rule that a stored stop could not be displaced: a balancing refresh landing
inside `offered_current_wait_time` after a pause now cancels that pause, an emergency pause among
them, because nothing on this path distinguishes one stop from another.

#### Scenario: a different current arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and a `cmd.smart_charge.set` for a different
  current arrives before the deadline
- **THEN** the write replaces the stop and is sent at the deadline; the charger is never paused

#### Scenario: a same current arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and a write for the last-sent current arrives
- **THEN** the write replaces the stop and is sent at the deadline, the dedup notwithstanding

#### Scenario: a stop replaces a stored current
- **WHEN** a dynamic-current write is stored in the slot and `cmd.charge.stop` arrives before the
  deadline
- **THEN** the stop replaces the stored write and is sent at the deadline

#### Scenario: an explicit start arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and `cmd.charge.start` arrives before the deadline
- **THEN** the start replaces the stop and the charger resumes at the deadline

#### Scenario: a stream of alternating kinds
- **WHEN** start, stop, start and stop arrive in turn inside one wait window
- **THEN** exactly one command reaches Easee at the deadline and it is the final stop

#### Scenario: the displacement is logged
- **WHEN** a `set_current 26` displaces a stored stop
- **THEN** `set_current 26 preempts stop` is logged for that charger

### Requirement: A Restart Is An Ordered Pair
A session restart SHALL occupy the channel as an ordered pair - a stop followed by a resume - stored
atomically, so that no command can interleave between the two halves. The resume SHALL be sent one
`offered_current_wait_time` after the stop, because Easee ignores a dynamic-current write that
reaches the charger within that window of a stop. A restart SHALL NOT be stored unless both halves
are known, so a restart can never end with the charger paused.

A later command SHALL discard the pair in whole, both the stop and the resume, and take the slot
itself. Preempting only one half would either pause the charger with no resume to follow, or resume
it at a current the hub has since superseded.

#### Scenario: an uncontested restart
- **WHEN** a restart is stored and nothing else arrives
- **THEN** the stop is sent at the deadline and the resume one wait time after it, leaving the
  charger charging

#### Scenario: a command preempts a stored restart
- **WHEN** a restart is stored and a dynamic-current write arrives before either half is sent
- **THEN** both halves are discarded and only that write is sent at the deadline

#### Scenario: a command arrives between the halves
- **WHEN** the stop has been sent and a dynamic-current write arrives before the resume's deadline
- **THEN** the write replaces the resume and is sent at that deadline

#### Scenario: the pair's first half is refused
- **WHEN** the stop of a restart is sent at once and Easee refuses it
- **THEN** the resume is discarded rather than sent on its own, and the slot is left free for the
  next command

#### Scenario: the thing is deleted with a restart stored
- **WHEN** a restart is stored and the thing is deleted before either half is sent
- **THEN** both halves are discarded and no call reaches Easee

### Requirement: Deferred Commands Do Not Outlive The Thing
The timer that sends a stored command SHALL be cancelled, and the slot cleared, when the charger's
connector is disconnected. `cmd.thing.delete` disconnects the thing, and a timer armed before it
would otherwise issue `StopCharging` or `UpdateDynamicCurrent` for a charger the hub no longer owns.
Uninstall is already safe because it clears the credentials first; thing delete is not.

#### Scenario: thing deleted with a command stored
- **WHEN** a command is stored in the slot and the thing is deleted before the deadline
- **THEN** the timer is cancelled and no call reaches Easee

#### Scenario: the timer fires after teardown cleared the slot
- **WHEN** the send fires for a slot that teardown has already cleared
- **THEN** it is a no-op

### Requirement: State Re-sync After Start Or Stop
After `cmd.charge.start` or `cmd.charge.stop` is sent to Easee and accepted, the adapter SHALL request a state re-sync for that charger 15 s after the send, because Easee's live stream may never deliver the state, power or phase-current change the command causes. At most one re-sync per charger SHALL be pending; a later accepted start or stop SHALL replace it with one 15 s after the later send. A command that fails, or that is stored for a later send, SHALL NOT schedule a re-sync until it is actually sent.

#### Scenario: stop accepted
- **WHEN** a pause-charging command is sent and Easee accepts it
- **THEN** a state re-sync for that charger is requested 15 s later

#### Scenario: start accepted
- **WHEN** a start or resume command is sent and Easee accepts it
- **THEN** a state re-sync for that charger is requested 15 s later

#### Scenario: second command within the delay
- **WHEN** another start or stop is accepted before the pending re-sync runs
- **THEN** only one re-sync runs, 15 s after the later send

#### Scenario: command fails
- **WHEN** the start or stop request fails
- **THEN** no re-sync is scheduled

#### Scenario: stop stored by the pacing window
- **WHEN** a stop is stored because a write went out within `offered_current_wait_time`
- **THEN** the re-sync is scheduled 15 s after the stored stop is actually sent

### Requirement: Charging Watchdog
The adapter SHALL check every minute each charger whose cached state is `charging`. When no `ChargerOPState` or `TotalPower` observation, the two the state report is built from, has arrived for that charger for 5 minutes, it SHALL request a state re-sync, and while the silence lasts it SHALL request at most one re-sync every 5 minutes. `DynamicChargerCurrent` observations SHALL NOT count as activity, because Easee echoes the adapter's own writes with them while the rest of the stream is silent.

#### Scenario: charging and silent
- **WHEN** a charger is cached as `charging` and no state or power observation arrived for 5 minutes
- **THEN** a state re-sync is requested for it

#### Scenario: only command echoes arrive
- **WHEN** the only observations for a charger cached as `charging` are `DynamicChargerCurrent`
- **THEN** the watchdog still requests a re-sync after 5 minutes

#### Scenario: not charging
- **WHEN** a charger's cached state is anything other than `charging`
- **THEN** the watchdog requests no re-sync however long the stream is silent

#### Scenario: silence continues
- **WHEN** the silence continues after a watchdog re-sync
- **THEN** the next re-sync comes no sooner than 5 minutes after the previous one

