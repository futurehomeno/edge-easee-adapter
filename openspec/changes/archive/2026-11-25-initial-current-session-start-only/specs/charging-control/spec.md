## MODIFIED Requirements

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
charged, as the Zaptec adapter does (issue 44, the operator, 2026-11-24: "when below minimum, charging
should start with initial_charging"). Otherwise: the cached requested offered current; if that is zero
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

## ADDED Requirements

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
