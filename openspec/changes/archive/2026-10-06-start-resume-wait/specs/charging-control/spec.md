## MODIFIED Requirements

### Requirement: Start Charging
`cmd.charge.start` SHALL resume the charger by writing a dynamic current, because the Easee resume
command clears the dynamic current. The starting current SHALL be chosen as follows: the cached
requested offered current; if that is zero or less — meaning unknown, either because no load balancer
ever set one or because the session-finished observation cleared it — the cached max current
instead; otherwise, for a non-slow mode, raised to at least `initial_charging_current` (default 16A).
Slow mode SHALL be exempt from that floor and SHALL instead use `slowChargingCurrentInAmperes` when
that is greater than zero, rounded to the nearest ampere. A resulting current of zero SHALL fail with
`invalid start current`. A start Easee accepts SHALL succeed even when the charger does not echo it
back within the current wait duration; the miss SHALL be logged at info. A paused charger can take
well over that wait to act on the resume, and waiting longer would hold the chargepoint service lock
that a stop or a lower current needs. The state report shows whether the charger resumed. A start the command channel defers SHALL
report success at once; the deferred send checks the echo.

#### Scenario: the charger never echoes the start back
- **WHEN** Easee accepts the start and no observation confirms the current within the wait duration
- **THEN** the command succeeds and the missing echo is logged at info

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

