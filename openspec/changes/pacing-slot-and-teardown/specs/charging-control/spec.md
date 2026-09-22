# Charging Control

## MODIFIED Requirements

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

## ADDED Requirements

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
