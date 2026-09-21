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

## ADDED Requirements

### Requirement: A Stored Stop Holds The Slot
A stop stored in the per-charger command channel SHALL NOT be displaced by any later
dynamic-current write, whatever current that write carries and whichever command issued it - an
explicit `cmd.charge.start` included. A stop is a safety command, an emergency pause among them,
and a balancing refresh landing inside `offered_current_wait_time` must not cancel it. Nothing on
this path distinguishes an operator's start from a balancer's resume, so the stop wins over both
rather than the adapter deciding a paused charger is safe to resume. The later write SHALL be
dropped, its command SHALL still report success, and the stop SHALL be sent at the deadline the
stop's own arrival set. A user who wants to charge after a deferred stop presses Start once the
stop has landed.

This supersedes the previous "latest wins across kinds" rule, under which a start behind a stored
stop replaced it and the charger was never paused.

#### Scenario: a different current arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and a `cmd.smart_charge.set` for a different
  current arrives before the deadline
- **THEN** the stop stays in the slot, the write is dropped, and the stop is sent at the deadline

#### Scenario: a same current arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and a write for the last-sent current arrives
- **THEN** the stop stays in the slot and is sent at the deadline

#### Scenario: a stop replaces a stored current
- **WHEN** a dynamic-current write is stored in the slot and `cmd.charge.stop` arrives before the
  deadline
- **THEN** the stop replaces the stored write and is sent at the deadline

#### Scenario: an explicit start arrives after a stop
- **WHEN** `cmd.charge.stop` is stored in the slot and `cmd.charge.start` arrives before the deadline
- **THEN** the start reports success, no resume reaches Easee, and the charger is paused at the
  deadline

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
