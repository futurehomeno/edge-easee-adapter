## MODIFIED Requirements

### Requirement: Start Charging
`cmd.charge.start` SHALL resume the charger by writing a dynamic current, because the Easee resume
command clears the dynamic current. A start with a mode SHALL be remembered as the session's mode,
replacing the one remembered before; a start without a mode SHALL be treated as the remembered mode,
because energy guard resumes a session it paused with a start that carries none. The session-finished
observation SHALL forget the mode, where it clears the cached requested offered current. The mode is
kept in memory only, so after an adapter restart a start without a mode is not slow. The starting
current SHALL be chosen as follows: the cached requested offered current; if that is zero or less — meaning unknown, either because no load balancer
ever set one or because the session-finished observation cleared it — the cached max current
instead; if that is also zero — meaning the cache has not been seeded by an observation since the
last restart — 32A, the same ceiling the offered-current write already falls back to, so a restart
does not leave the charger unstartable until observation 47 arrives; otherwise, for every start that
is not slow mode, with or without a mode, raised to at least `initial_charging_current` (default
16A), because some cars need the higher current to start; the balancer adjusts it afterwards. The
charger's max current SHALL cap the starting current.
Slow mode SHALL be exempt from that floor and SHALL instead use `slowChargingCurrentInAmperes` when
that is greater than zero, rounded to the nearest ampere. A resulting current of zero SHALL fail with
`invalid start current`. A start Easee accepts SHALL succeed even when the charger does not echo it
back within the current wait duration; the miss SHALL be logged at info. A paused charger can take
well over that wait to act on the resume, and waiting longer would hold the chargepoint service lock
that a stop or a lower current needs. The state report shows whether the charger resumed.

#### Scenario: the charger never echoes the start back
- **WHEN** Easee accepts the start and no observation confirms the current within the wait duration
- **THEN** the command succeeds and the missing echo is logged at info

#### Scenario: the start is deferred
- **WHEN** the start arrives within `offered_current_wait_time` of another command
- **THEN** the command succeeds without waiting for an echo, and the write goes out at the deadline

#### Scenario: normal start with a cached current below the floor
- **WHEN** the cached requested offered current is 10A and the initial charging current is 16A
- **THEN** the charger is started at 16A

#### Scenario: start without a mode after a cut
- **WHEN** a start without a mode arrives, the cached requested offered current is 7A and the initial
  charging current is 16A
- **THEN** the charger is started at 16A

#### Scenario: start floor capped by the charger maximum
- **WHEN** a start without a mode arrives, the cached requested offered current is 7A, the initial
  charging current is 16A and the charger's max current is 10A
- **THEN** the charger is started at 10A

#### Scenario: slow start
- **WHEN** the mode is `slow` and `slowChargingCurrentInAmperes` is 6
- **THEN** the charger is started at 6A, without the initial-charging-current floor

#### Scenario: slow start with no slow current configured
- **WHEN** the mode is `slow` and `slowChargingCurrentInAmperes` is 0
- **THEN** the cached requested offered current is used unchanged

#### Scenario: slow session resumed without a mode
- **WHEN** a session was started in mode `slow`, energy guard stops it and starts it again without a
  mode, and `slowChargingCurrentInAmperes` is 6
- **THEN** the charger is started at 6A

#### Scenario: slow session resumed without a mode, no slow current configured
- **WHEN** a session was started in mode `slow` at a cached current of 6A, energy guard stops it and
  starts it again without a mode, and `slowChargingCurrentInAmperes` is 0
- **THEN** the charger is started at 6A, without the initial-charging-current floor

#### Scenario: normal session resumed without a mode
- **WHEN** a session was started in mode `normal`, the balancer cut it to 7A, energy guard stops it and
  starts it again without a mode
- **THEN** the charger is started at 16A

#### Scenario: start without a mode after a slow session ended
- **WHEN** a session was started in mode `slow`, the session-finished observation arrives, and a
  start without a mode follows
- **THEN** the start is not slow and gets the initial-charging-current floor

#### Scenario: explicit normal start after a slow start
- **WHEN** a session was started in mode `slow` and a start in mode `normal` follows at a cached
  current of 6A
- **THEN** the charger is started at 16A, and a later start without a mode is not slow

#### Scenario: no cached current
- **WHEN** the cached requested offered current is zero or negative
- **THEN** the cached max current is used as the starting current

#### Scenario: nothing to start from
- **WHEN** both the cached requested offered current and the cached max current are zero
- **THEN** the charger is started at 32A rather than failing with `invalid start current`, because
  an unseeded cache after a restart is not the same as a charger that cannot charge
