## MODIFIED Requirements

### Requirement: Offered Current
`cmd.current_session.set_current` SHALL write the dynamic current through the per-charger command
channel. The requested value SHALL be clamped to the cached max current, or to 32A when no max
current is cached, logging the clamp when the requested value or the limit differs from the
charger's previous clamp, and not on a repeat: Energy Guard resends its target current on every
balancing tick. A `cmd.current_session.set_current` that is not clamped SHALL reset this, so the
next clamp is logged; a start or a phase-mode restart SHALL NOT. Unless forced, a value equal to the
last requested value within `offered_current_wait_time` SHALL be skipped. When the write goes out
immediately, the requested value SHALL be cached with the current hub time and the adapter SHALL
wait up to `current_wait_duration` (default 3s) for the charger to echo it back over SignalR; a
deferred write does both when it is sent. The wait SHALL resolve on either the value delivered by
the notification or the value the cache holds when it wakes: notifications are dropped on a full
listener buffer, so neither source alone sees every echo - a confirmation immediately superseded by
a further observation is gone from the cache, while a delivered value can itself be the stale one.

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
- **WHEN** 20A is requested at a 13A max, charging is started, and 20A is requested again
- **THEN** the clamp is logged once

#### Scenario: clamp changes
- **WHEN** 20A is requested at a 13A max, then 22A at a 13A max, then 22A at a 16A max
- **THEN** the clamp is logged for each of the three requests

#### Scenario: clamp after an unclamped request
- **WHEN** 22A is requested at a 16A max, then 16A, then 22A again
- **THEN** the clamp is logged for both 22A requests
