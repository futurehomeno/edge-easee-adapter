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
command clears the dynamic current. A start SHALL be a session start when the charger has not been
observed charging since the car was connected or since the previous session ended (the
session-finished observation); a `set_current` before it SHALL NOT make it a resume. Every later
start in the same session SHALL be a resume. Whether the session has charged is kept in memory: after
an adapter restart the replayed charger state sets it again when the charger is charging.

The starting current SHALL be chosen as follows. While the adapter's own pause for a sub-minimum
offered current is remembered (Sub-Minimum Current Pauses), or when the cached requested current is
below `min_charging_current` (a value kept under a user stop), a start that is not slow SHALL start at
the greater of `initial_charging_current` and `min_charging_current`, whether or not the session has
charged, as the Zaptec adapter does (issue 44, the operator, 2026-11-24: "when below minimum, chargins
hould start with initial_charging"). Otherwise: the cached requested offered current; if that is zero
or less — meaning unknown, either because no load balancer ever set one or because the
session-finished observation cleared it — the cached max current instead, so installs without a power
manager start at full current; if that is also zero — meaning the cache has not been seeded by an
observation since the last restart — 32A, the same ceiling the offered-current write already falls
back to, so a restart does not leave the charger unstartable until observation 47 arrives. A session
start that is not slow SHALL raise a cached requested current to at least `initial_charging_current`
(default 16A), because some cars need the higher current to start; the balancer adjusts it
afterwards. A resume SHALL start at the cached requested current, the current in force before the
pause, and SHALL NOT raise it to `initial_charging_current` (the operator, 2026-11-24: "initial
current is for session start, not resume"), so a load balancer's resume does not start above the
current it had balanced the charger to. This reverses the decision of 2026-11-23 that every start
that is not slow starts at the initial charging current (issue 42). The charger's max current SHALL
cap the starting current, 32A when it is not known. A start SHALL NOT otherwise check `min_charging_current`.

A start without a mode SHALL keep the mode of the session it resumes: after a slow start it is not
raised, until a start with a mode arrives or the session ends and the cached requested offered
current is cleared. This memory is not kept across an adapter restart.
Slow mode SHALL be exempt from the initial charging current and SHALL instead use
`slowChargingCurrentInAmperes` when that is greater than zero, rounded to the nearest ampere. A
resulting current of zero SHALL fail with `invalid start current`. A start Easee accepts SHALL succeed
even when the charger does not echo it back within the current wait duration; the miss SHALL be
logged at info. A paused charger can take well over that wait to act on the resume, and waiting
longer would hold the chargepoint service lock that a stop or a lower current needs. The state report
shows whether the charger resumed.

#### Scenario: the charger never echoes the start back
- **WHEN** Easee accepts the start and no observation confirms the current within the wait duration
- **THEN** the command succeeds and the missing echo is logged at info

#### Scenario: the start is deferred
- **WHEN** the start arrives within `offered_current_wait_time` of another command
- **THEN** the command succeeds without waiting for an echo, and the write goes out at the deadline

#### Scenario: normal start with a cached current below the floor
- **WHEN** the charger has not charged since the car was connected, a `set_current` of 8A was made
  and a start arrives, with or without a mode, with an initial charging current of 16A
- **THEN** the charger is started at 16A, a session start

#### Scenario: start floor capped by the charger maximum
- **WHEN** the session has not charged, a start without a mode arrives, the cached requested offered
  current is 7A, the initial charging current is 16A and the charger's max current is 10A
- **THEN** the charger is started at 10A

#### Scenario: start without a mode after a cut
- **WHEN** a start without a mode arrives, the cached requested offered current is 7A and the initial
  charging current is 16A
- **THEN** the charger is started at 16A when the session has not charged, a session start, and at
  7A when it has, a resume

#### Scenario: no cached current
- **WHEN** the cached requested offered current is zero or negative and the charger's max current is
  20A
- **THEN** the charger is started at 20A, the cached max current, whether or not the session has
  charged

#### Scenario: resume after an energy-guard pause
- **WHEN** the charger charged in this session, energy guard set 7A, stopped the charger and sends a
  start without a mode, with an initial charging current of 16A
- **THEN** the charger is started at 7A

#### Scenario: resume after a user stop
- **WHEN** the charger charged in this session at 10A, the user stopped it and starts it in mode
  `normal`
- **THEN** the charger is started at 10A

#### Scenario: resume capped by the charger maximum
- **WHEN** the session charged at 20A and the charger's max current is now 10A
- **THEN** the resume starts at 10A

#### Scenario: the next session starts again at the floor
- **WHEN** the charger charged, the session ended, a new car was connected, a `set_current` of 8A was
  made and a start arrives
- **THEN** the charger is started at 16A

#### Scenario: start after the adapter's own pause
- **WHEN** the adapter paused the charger for a value below `min_charging_current` (6A), the value
  before the pause was 10A or 20A, the initial charging current is 16A and the charger's max current
  is 32A
- **THEN** the start writes 16A, whether or not the session had charged

#### Scenario: start after a user stop that kept a sub-minimum value
- **WHEN** the session charged, the user stopped it, a `set_current` of 3A was kept and a start in
  mode `normal` arrives
- **THEN** the charger is started at 16A; with `min_charging_current` 20A and 12A kept, at 20A

#### Scenario: start after the adapter's own pause with a raised minimum
- **WHEN** `min_charging_current` is 10A, the initial charging current is 16A, the charger's max
  current is 8A and the adapter paused the charger
- **THEN** the start writes 8A

#### Scenario: start without a mode in a slow session
- **WHEN** the session was started in mode `slow`, the cached requested offered current is 6A and a
  start without a mode arrives
- **THEN** the charger is started at 6A

#### Scenario: start without a mode after the slow session ended
- **WHEN** the slow session ended, a new session that has not charged was started without a mode,
  the cached requested offered current is 7A and a start without a mode arrives
- **THEN** the charger is started at 16A

#### Scenario: the session after a slow one begins with a current
- **WHEN** the slow session ended, the next session's first command is a `set_current` of 7A, the
  session has not charged, and a start without a mode arrives
- **THEN** the charger is started at 16A

#### Scenario: slow start
- **WHEN** the mode is `slow` and `slowChargingCurrentInAmperes` is 6
- **THEN** the charger is started at 6A, without the initial charging current

#### Scenario: slow start with no slow current configured
- **WHEN** the mode is `slow` and `slowChargingCurrentInAmperes` is 0
- **THEN** the cached requested offered current is used unchanged

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
on stop and ignores a dynamic-current write that follows it too closely. It SHALL mark the charger
user-stopped (A User Stop Holds Until A Start).

#### Scenario: stop after a quiet period
- **WHEN** no command was sent for the charger within `offered_current_wait_time`
- **THEN** the pause-charging command is sent and a non-202 response is reported as an error

#### Scenario: stop too soon after a current change
- **WHEN** a dynamic-current write went out within `offered_current_wait_time` before the stop
- **THEN** the stop is stored rather than refused, and sent at the deadline

#### Scenario: write shortly after a stop
- **WHEN** a `cmd.charge.start` arrives within `offered_current_wait_time` of a stop
- **THEN** its write is stored and sent at the deadline the stop opened

### Requirement: Offered Current
`cmd.current_session.set_current` SHALL write the dynamic current through the per-charger command
channel. The requested value SHALL be clamped to the cached max current, or to 32A when no max
current is cached, logging the clamp when the requested value or the limit differs from the
charger's previous clamp, and not on a repeat: Energy Guard resends its target current on every
balancing tick. A `cmd.current_session.set_current` that is not clamped SHALL reset this, so the
next clamp is logged; a start or a phase-mode restart SHALL clamp without logging and SHALL NOT
reset it. Unless forced, a value equal to the last requested value within
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

#### Scenario: repeated clamp
- **WHEN** 20A is requested several times while the cached max current stays 13A
- **THEN** the clamp is logged once and every request is clamped to 13A

#### Scenario: start between repeated clamps
- **WHEN** 20A is requested at a 13A max, charging is started at a cached 16A, and 20A is requested
  again
- **THEN** the start is clamped to 13A without a log line and the clamp is logged once

#### Scenario: clamp changes
- **WHEN** 20A is requested at a 13A max, then 22A at a 13A max, then 22A at a 16A max
- **THEN** the clamp is logged for each of the three requests

#### Scenario: clamp after an unclamped request
- **WHEN** 22A is requested at a 16A max, then 16A, then 22A again
- **THEN** the clamp is logged for both 22A requests

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
The Easee command endpoints for dynamic current, max current and stop charging SHALL be
treated as successful only on HTTP 202. The set-phase-mode and cable-lock endpoints SHALL accept both 200 and 202,
because Easee has been observed answering set-phase-mode, documented as 200, with 202, and
cable-lock, expected to answer 202, with 200. Any other status SHALL be logged
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
dynamic-current write SHALL displace a stored stop. A `cmd.current_session.set_current` under a user
stop SHALL NOT reach the channel (A User Stop Holds Until A Start), so a user's stop is displaced
only by `cmd.charge.start`; the stops a value displaces are the adapter's own sub-minimum pause and
the first half of a restart. The displaced command SHALL NOT reach Easee, its own command having
already reported success, and the deadline SHALL NOT move. The dedup that suppresses a repeated
current SHALL NOT apply while a stop is stored, so a write carrying the last-sent value displaces
that stop like any other.

Displacing a stored command SHALL be logged as `[<id>] <new> preempts <old>`, naming both, so the
log shows which command was dropped and what replaced it rather than only what was stored. When the
displaced command is a pair, both halves SHALL be named, or a discarded resume would leave no trace.

The charger acts on whatever it last heard, so the hub's latest intent is the only one worth
sending; holding an older command in the slot makes the adapter answer a command it did not carry
out. This supersedes the rule that a balancing refresh inside `offered_current_wait_time` after a
user's pause cancels that pause (issue 33).

#### Scenario: a different current arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and a `cmd.current_session.set_current` for a
  different current arrives before the deadline
- **THEN** the stop stays stored and is sent at the deadline; the value is only kept for the next
  `cmd.charge.start`

#### Scenario: a same current arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and a write for the last-sent current arrives
- **THEN** the stop stays stored and is sent at the deadline, and nothing else is sent

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
- **WHEN** a `set_current 26` displaces the adapter's own stored sub-minimum pause
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
The adapter SHALL check every minute each charger whose cached state is `charging`. When no `ChargerOPState` observation and no `TotalPower` observation above zero, the two the state report is built from, has arrived for that charger for 5 minutes, it SHALL request a state re-sync, and while the silence lasts it SHALL request at most one re-sync every 5 minutes. `DynamicChargerCurrent` observations SHALL NOT count as activity, because Easee echoes the adapter's own writes with them while the rest of the stream is silent.

#### Scenario: charging and silent
- **WHEN** a charger is cached as `charging` and no state or power observation arrived for 5 minutes
- **THEN** a state re-sync is requested for it

#### Scenario: only command echoes arrive
- **WHEN** the only observations for a charger cached as `charging` are `DynamicChargerCurrent`
- **THEN** the watchdog still requests a re-sync after 5 minutes

#### Scenario: only zero power arrives
- **WHEN** the only observations for a charger cached as `charging` are `TotalPower` of zero
- **THEN** the watchdog still requests a re-sync after 5 minutes

#### Scenario: not charging
- **WHEN** a charger's cached state is anything other than `charging`
- **THEN** the watchdog requests no re-sync however long the stream is silent

#### Scenario: silence continues
- **WHEN** the silence continues after a watchdog re-sync
- **THEN** the next re-sync comes no sooner than 5 minutes after the previous one

### Requirement: Teardown Waits Out Immediate Sends
Teardown SHALL wait for a start, stop or dynamic-current write that is being sent at once until it
reaches Easee, and SHALL cancel the pending re-sync. A command Easee answers after Teardown, a phase
mode among them, SHALL arm no re-sync.

#### Scenario: thing deleted during an immediate stop
- **WHEN** the thing is deleted while a stop sent at once is still waiting for Easee
- **THEN** Teardown returns only after the stop finished, and no re-sync fires afterwards

#### Scenario: thing deleted during a phase mode set
- **WHEN** the thing is deleted while a set-phase-mode call is still waiting for Easee
- **THEN** no re-sync fires afterwards

#### Scenario: command after Teardown
- **WHEN** a start, stop or dynamic-current write for the charger arrives after Teardown
- **THEN** it never reaches Easee and arms no re-sync

### Requirement: Sub-Minimum Current Pauses
A `cmd.current_session.set_current` below `min_charging_current` (default 6 A), 0 A included, SHALL
pause the charger with the Easee pause-charging command, through the per-charger command channel
like `cmd.charge.stop`, and the adapter SHALL remember that pause as its own. The next value at or
above the minimum SHALL be written as the dynamic current with the recent-value dedup bypassed, which
resumes the charger, and SHALL clear that memory. A `cmd.charge.stop` SHALL
clear it too: under a user stop no value resumes the charger (A User Stop Holds Until A Start).

The framework rejects a current under 6 A before the controller runs, so a value from 0 to 5 A SHALL
be handed to the controller in routing before the framework's handler: on success the message
interface SHALL be renamed with a `.paused` suffix so the framework's handler skips it, and the
adapter SHALL send the current-session report itself. A value from 6 A up to the minimum reaches the
controller through the framework and pauses there. A max current is still checked against 6 A.

#### Scenario: an energy manager sets 0 A
- **WHEN** `cmd.current_session.set_current` arrives with a value of 0
- **THEN** the pause-charging command is sent and no dynamic current is written
- **AND** the message interface is renamed with a `.paused` suffix and a current-session report is sent

#### Scenario: a current below a raised minimum
- **WHEN** `min_charging_current` is 10 and `cmd.current_session.set_current` arrives with 8
- **THEN** the charger is paused and no dynamic current is written

#### Scenario: the next valid value resumes the adapter's pause
- **WHEN** the adapter paused the charger for a sub-minimum value and a value at or above the minimum
  arrives, equal to the last one written inside `offered_current_wait_time`
- **THEN** the dynamic current is written anyway, resuming the charger

#### Scenario: a pause arrives while a valid value is being written
- **WHEN** a value at or above the minimum is still on its way to Easee and a sub-minimum value arrives
- **THEN** the pause stays the adapter's own once the write returns, and the next valid value resumes it

#### Scenario: a valid value arrives while the pause is being sent
- **WHEN** a sub-minimum value's pause is still on its way to Easee and a value at or above the
  minimum arrives, equal to the last one written inside `offered_current_wait_time`
- **THEN** the value is written anyway, resuming the charger

#### Scenario: a user stop is not resumed by the adapter
- **WHEN** the adapter paused the charger, then `cmd.charge.stop` arrives, then a value at or above
  the minimum equal to the last one written inside `offered_current_wait_time`
- **THEN** nothing is sent, and the value is kept only for the next `cmd.charge.start`

#### Scenario: a current under 6 A at a lowered minimum
- **WHEN** `min_charging_current` is 1 and `cmd.current_session.set_current` arrives with 3
- **THEN** 3 A is written as the dynamic current and the message interface is renamed with a `.paused` suffix

#### Scenario: the pause fails
- **WHEN** handing a value under 6 A to the controller returns an error
- **THEN** the message is left unrenamed and the framework handles it normally

#### Scenario: the pause is sent but the session report fails
- **WHEN** the pause for 0 A succeeds and the current-session report cannot be sent
- **THEN** the caller is answered with an error report

#### Scenario: a negative value
- **WHEN** the value is below zero
- **THEN** the message is left untouched

### Requirement: A User Stop Holds Until A Start
`cmd.charge.stop` SHALL mark the charger user-stopped, with the hub time of the stop, in the
adapter's database, so the mark survives an adapter restart. While it is marked,
`cmd.current_session.set_current` SHALL send nothing to Easee, a sub-minimum value included, and
SHALL only keep the value as the requested current, so the next `cmd.charge.start` starts at it (a
sub-minimum one at the initial charging current, Start Charging); the
adapter's own pause stays unmarked. A phase mode change SHALL NOT bounce the session while it is
marked. Easee resumes a stopped charger on any dynamic-current write, so only `cmd.charge.start`
may start charging after a user stop (the operator, 2026-11-23).

The mark SHALL end on `cmd.charge.start`, and SHALL be put back when the start's write fails. A
start stored in the channel has already been answered as accepted: its later failure is only logged
and the mark stays ended, so the next value resumes the charger as the start asked. It
SHALL also end on a charger state observation timestamped after the stop that is either
disconnected (the car unplugged) or a change into charging (started from the Easee app, for
example). Every other state SHALL keep it. An observation timestamped before the stop describes the
charger before Easee acted on the stop and SHALL NOT end it.

#### Scenario: a value after a user stop
- **WHEN** `cmd.charge.stop` was sent and `cmd.current_session.set_current` arrives with a value at
  or above the minimum
- **THEN** nothing is sent and a later `cmd.charge.start` starts at that value

#### Scenario: a sub-minimum value under a user stop
- **WHEN** `cmd.charge.stop` was sent and a value below the minimum arrives
- **THEN** nothing is sent, and a later valid value sends nothing either

#### Scenario: a start lifts the stop
- **WHEN** `cmd.charge.stop` was sent, then `cmd.charge.start`, then a new value
- **THEN** the start resumes the charger and the value is written as usual

#### Scenario: the start fails
- **WHEN** Easee refuses the write of a `cmd.charge.start` after a user stop
- **THEN** the charger stays marked user-stopped

#### Scenario: the car is unplugged
- **WHEN** the charger reports disconnected after the stop
- **THEN** the mark ends and the next value is written as usual

#### Scenario: charging started elsewhere
- **WHEN** the charger reports a change into charging, timestamped after the stop
- **THEN** the mark ends

#### Scenario: a charging observation older than the stop
- **WHEN** a charging observation timestamped before the stop arrives after it
- **THEN** the mark holds

#### Scenario: Other states keep a user stop
- **WHEN** the charger reports error, completed, await_auth, de_auth, unknown or offline after the
  stop
- **THEN** the mark holds

#### Scenario: an adapter restart keeps the stop
- **WHEN** the adapter restarts after `cmd.charge.stop` and a value arrives
- **THEN** nothing is sent

#### Scenario: a stop between a value's check and its write
- **WHEN** a value under 6 A, outside the chargepoint service lock, passes the user-stop check and
  `cmd.charge.stop` lands before its write takes the channel
- **THEN** the write is not sent and the stop stays the last command

#### Scenario: a stop made while an observation ends an older one
- **WHEN** a charger state that ends the user stop is being handled and `cmd.charge.stop` lands
  meanwhile
- **THEN** the new stop holds

#### Scenario: a phase mode change under a user stop
- **WHEN** the charger still reports charging after `cmd.charge.stop` and a new phase mode is set
- **THEN** the mode is sent but the session is not bounced

### Requirement: A Session End Resets The Session Values
A charger state observation that ends the session — the one that clears the cached requested offered
current: disconnected, completed, error, offline, unknown, awaiting authentication or
de-authenticating — SHALL return every value the adapter keeps for that session to its default (the
operator, 2026-11-24: "Unplugging is session finished so session specific values should be reset to
default"): the cached requested offered current, whether the session charged, the adapter's own
pause (Sub-Minimum Current Pauses) and the slow-session memory (Start Charging). The user stop (A
User Stop Holds Until A Start) and the `use_locked_3_phases` lock end when the car is disconnected.

When the car is disconnected, a write or a phase bounce still waiting in the command channel SHALL be
dropped: sent later, it would put the finished session's current back into the cache for the next
car. A charger that reports any other session end keeps the car, so a waiting command, a cut or a
bounce's resume, SHALL still be sent. A write already being sent cannot be called back, but it SHALL NOT put
its current back into the cache: it is stamped with the time it left, older than the session end.
The slow-session memory SHALL be forgotten at the session end itself, even when a write still waiting
puts a current back into the cache afterwards.
A pause or a resume whose command fails while the session ends SHALL NOT put back the adapter's own
pause the session end forgot, and a start being sent while it ends SHALL NOT put back the slow-start
memory.

#### Scenario: a session end resets the session values
- **WHEN** the session charged at 10A, the adapter paused the charger for a value below the minimum,
  the car is unplugged, and a start without a mode arrives for the next car
- **THEN** the start is a session start with nothing cached: the charger is started at its max
  current, not at the initial charging current

#### Scenario: the adapter's pause ends with an offline session end
- **WHEN** the adapter paused the charger and the charger reports offline
- **THEN** the pause is forgotten, and a later start with nothing cached starts at the max current

#### Scenario: an unplugged car drops a waiting write
- **WHEN** a write of 7A waits in the command channel and the car is unplugged
- **THEN** the write is not sent

#### Scenario: an unplugged car drops a waiting phase bounce
- **WHEN** a phase bounce's stop and resume wait in the command channel and the car is unplugged
- **THEN** neither is sent

#### Scenario: an offline charger still gets the waiting write
- **WHEN** a write of 7A waits in the command channel and the charger reports offline
- **THEN** the write is sent at its deadline

#### Scenario: an unplugged car during a write being sent
- **WHEN** a write of 7A is being sent, the car is unplugged before Easee answers, and a start without a
  mode arrives for the next car
- **THEN** the cached requested current stays cleared and the charger is started at its max current

#### Scenario: a command failing across the unplug
- **WHEN** the adapter paused the charger, a resume or another sub-minimum value is being sent, the car
  is unplugged and the command fails, and a start without a mode arrives for the next car
- **THEN** the charger is started at its max current, not at the initial charging current

#### Scenario: a session end during a slow start being sent
- **WHEN** a start in mode `slow` is being sent, the charger reports completed before Easee answers, a
  waiting write puts 7A into the cache, and a start without a mode arrives before the session charges
- **THEN** the start is a session start: the charger is started at 16A

#### Scenario: a session end forgets the slow start
- **WHEN** the session was started in mode `slow`, a write of 7A waits in the command channel, the
  charger reports completed, the write is sent, and a start without a mode arrives before the
  session charges again
- **THEN** the start is a session start: the charger is started at 16A

### Requirement: Easee HTTP Requests
Every call to the Easee REST API other than the login and the token refresh SHALL carry the header
`Authorization: Bearer <access token>` and go to the configured `easee_base_url`. The commands SHALL be sent as:
- dynamic (offered) current: `POST /api/chargers/{id}/settings` with `{"dynamicChargerCurrent": <A>}`;
- max current: `POST /api/chargers/{id}/settings` with `{"maxChargerCurrent": <A>}`;
- stop: `POST /api/chargers/{id}/commands/pause_charging` with no body and no content type;
- cable lock: `POST /api/chargers/{id}/commands/lock_state` with `{"state": <bool>}`;
- phase mode: `POST /api/chargers/{id}/commands/set_phase_mode` with `{"phaseMode": <1|2|3>}`.

A command with a body SHALL carry the content type `application/*+json`. The reads SHALL be `GET /api/chargers`,
`GET /api/chargers/{id}/config`, `GET /api/chargers/{id}/site`,
`GET /api/chargers/{id}/details?alwaysGetChargerAccessLevel=false` and the health check `GET /health`; each SHALL
succeed only on 200, and any other status SHALL be logged with the response body and returned as an error naming
the status code. A read whose body does not decode SHALL fail.

#### Scenario: offered current written
- **WHEN** 10 A is written as the dynamic current of charger `XX12345`
- **THEN** `POST /api/chargers/XX12345/settings` is sent with `{"dynamicChargerCurrent":10}` and the bearer token

#### Scenario: stop sent
- **WHEN** a stop is sent for charger `XX12345`
- **THEN** `POST /api/chargers/XX12345/commands/pause_charging` is sent with no body

#### Scenario: read answered with an error
- **WHEN** `GET /api/chargers/XX12345/config` is answered with 500
- **THEN** the body is logged and an error naming status code 500 is returned

