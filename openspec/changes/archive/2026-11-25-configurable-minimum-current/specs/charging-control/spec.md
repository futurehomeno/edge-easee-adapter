## ADDED Requirements

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

## MODIFIED Requirements

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
