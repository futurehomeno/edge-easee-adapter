## ADDED Requirements

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
