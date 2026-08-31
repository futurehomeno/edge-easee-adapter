# Reporting Specification

## Purpose
Publish charger telemetry to FIMP: operating state, electricity meter values, alarms, and charging
session history. Reports are driven by SignalR observations as they arrive, and by a periodic polling
task as a backstop.

## Requirements

### Requirement: Charger State Report
`evt.chargepoint.state_report` SHALL report the cached charger state, except that a cached total
power greater than zero SHALL be reported as `charging` regardless of the cached state. The
chargepoint service SHALL advertise in `sup_states` the FIMP states `unknown`, `disconnected`,
`ready_to_charge`, `charging`, `finished`, `error`, `suspended_by_ev` and `requesting` — Easee's nine
operating states mapped to FIMP and deduplicated, `offline` and `de-authenticating` both collapsing
to `unknown`. The FIMP names are not the Easee ones: Easee's awaiting start maps to
`ready_to_charge`, while Easee's ready to charge maps to `suspended_by_ev`.

#### Scenario: power observed while state says otherwise
- **WHEN** the cached total power is above zero
- **THEN** `charging` is reported

#### Scenario: no power observed
- **WHEN** the cached total power is zero or unset
- **THEN** the cached charger state is reported as-is

#### Scenario: supported states advertised
- **WHEN** the chargepoint service specification is built
- **THEN** each distinct FIMP state appears once in `sup_states`

### Requirement: Charger State Observation Handling
A charger operating state observation SHALL update the cache under the timestamp guard, log at info
level when the mapped FIMP state changed and at debug level when it did not, and publish an unforced
state report. It SHALL set the state-online flag from whether the state is offline. A state meaning
the session has finished SHALL clear the cached requested offered current.

#### Scenario: session finishes
- **WHEN** the observation reports a session-finished state
- **THEN** the cached requested offered current is set to zero

#### Scenario: state transition
- **WHEN** the mapped FIMP state differs from the cached one
- **THEN** the transition is logged at info level and an unforced state report is published

### Requirement: Electricity Meter Reports
The meter service SHALL support the `W` and `kWh` units and SHALL be configured to report at least
every minute. `W` SHALL be answered from the cached total power. `kWh` SHALL be answered from the
cached lifetime energy, and SHALL fail with `energy value not updated` when no lifetime energy has
ever been observed. Any other unit SHALL fail with `unsupported unit: <unit>`.

#### Scenario: power requested
- **WHEN** a meter report is requested for `W`
- **THEN** the cached total power is returned

#### Scenario: energy never observed
- **WHEN** a meter report is requested for `kWh` and no lifetime energy has been observed
- **THEN** the report fails with `energy value not updated`

#### Scenario: unsupported unit
- **WHEN** a meter report is requested for any other unit
- **THEN** the report fails with `unsupported unit`

### Requirement: Extended Meter Report
The meter service SHALL advertise the extended values `current_phase1`, `current_phase2`,
`current_phase3`, `energy_import` and `power_import`, answering each from the cache. `energy_import`
SHALL be omitted from the report when no lifetime energy has been observed. Requested values with no
mapping SHALL be omitted silently.

#### Scenario: all values requested
- **WHEN** an extended report is requested for every advertised value and all are cached
- **THEN** each requested value appears in the report

#### Scenario: lifetime energy never observed
- **WHEN** an extended report including `energy_import` is requested and no lifetime energy exists
- **THEN** the report omits `energy_import` and still carries the other requested values

### Requirement: Power Observation
A total-power observation SHALL be cached in watts, converting from the kilowatts Easee reports by
multiplying by 1000. A successful cache update SHALL publish an unforced `W` meter report followed by
an unforced extended report for `power_import`.

#### Scenario: power observation arrives
- **WHEN** a total-power observation of 7.4 arrives
- **THEN** 7400 is cached and the two unforced meter reports are published

### Requirement: Phase Current Observations
The in-current T3, T4 and T5 observations SHALL populate the phase 1, 2 and 3 currents respectively,
each publishing its own unforced extended meter report on a successful cache update.

#### Scenario: phase current observed
- **WHEN** an in-current T3 observation arrives newer than the cached value
- **THEN** the phase 1 current is cached and an unforced extended report is published

### Requirement: Alarm Reports
The alarm service SHALL advertise the grounding fault, grid type fault and other charge error events.
An alarm report SHALL carry status `activate` when the event is cached as active and `deactivate`
otherwise.

#### Scenario: active alarm
- **WHEN** a report is requested for an event cached as active
- **THEN** the report carries status `activate`

#### Scenario: inactive alarm
- **WHEN** a report is requested for an event that is not cached as active
- **THEN** the report carries status `deactivate`

#### Scenario: error code observation
- **WHEN** an error-code observation arrives
- **THEN** the other-charge-error alarm is reported according to whether a fault is present

### Requirement: Charging Session Storage
Charging sessions SHALL be persisted per charger in a buntdb bucket named
`charging-sessions:<chargerID>`, keyed by the Easee session ID. Registering a session start SHALL
first close a previous session left open, by stamping its stop time with the new session's start
time. Registering a session stop SHALL write the session's start, stop and energy.

#### Scenario: previous session left open
- **WHEN** a session start is registered and the latest stored session has a zero stop time
- **THEN** that session's stop is set to the new session's start before the new one is stored

#### Scenario: session stop
- **WHEN** a session stop observation arrives
- **THEN** the session's start, stop and energy are written under its ID

#### Scenario: session observations publish a report
- **WHEN** either a session start or a session stop is registered successfully
- **THEN** an unforced current-session report is published

### Requirement: Session History Retrieval
The two most recent sessions SHALL be retrieved by parsing the bucket's keys as int64 session IDs,
sorting them descending, and reading the first two. A key that cannot be parsed as an int64 SHALL be
logged as an error and skipped.

#### Scenario: two sessions stored
- **WHEN** the history is read for a charger with several stored sessions
- **THEN** the two highest session IDs are returned, latest first

#### Scenario: unparsable key
- **WHEN** a bucket contains a key that is not an integer
- **THEN** the key is logged and skipped and the remaining keys are still read

### Requirement: Current Session Report
`evt.chargepoint.current_session_report` SHALL carry the cached session energy. When a stored session
exists it SHALL also carry that session's start and stop times, and the previous session's energy
when a second stored session exists. While the latest session is still open — its stop time is zero —
the report SHALL additionally carry the offered current, taken from the cache and capped at the
cached max current when one is known.

#### Scenario: open session
- **WHEN** the latest stored session has a zero stop time
- **THEN** the report carries the offered current, capped at the cached max current

#### Scenario: finished session
- **WHEN** the latest stored session has a stop time
- **THEN** the report carries its start and stop times and no offered current

#### Scenario: previous session known
- **WHEN** a second stored session exists
- **THEN** its energy is reported as the previous session energy

### Requirement: Lifetime Energy Rate Limiting
Lifetime energy observations SHALL be throttled in two stages, so that a charger streaming
continuous energy updates does not flood FIMP. First, an observation whose timestamp truncated to
the hour is not after the last reading's truncated timestamp SHALL be dropped before it is enqueued.
Observations that pass SHALL be enqueued, and the manager SHALL emit at most one report per
`energyLifetimeInterval` (default 10s). The hour boundary is therefore the binding limit in practice.

#### Scenario: same-hour observation
- **WHEN** a lifetime-energy observation arrives in the same hour as the last reading
- **THEN** it is dropped before reaching the interval timer

#### Scenario: new-hour observation
- **WHEN** a lifetime-energy observation arrives in a later hour than the last reading
- **THEN** it is enqueued and reported subject to the configured interval

### Requirement: Periodic Reporting Task
A car-charger reporting task SHALL run on `polling_interval` (default 10m) and SHALL be gated on the
app being connected, so that polling stops while the app reports a disconnected state.

#### Scenario: app connected
- **WHEN** the polling interval elapses and the lifecycle reports connected
- **THEN** the chargepoint, meter and alarm reports are refreshed

#### Scenario: app disconnected
- **WHEN** the lifecycle reports disconnected
- **THEN** the reporting task does not run

### Requirement: Connectivity And Ping
The thing connector SHALL report connectivity as an indirect cloud connection, up only when the
SignalR manager considers the charger connected. A ping SHALL succeed only when both the Easee HTTP
health endpoint answers and the manager considers the charger connected.

#### Scenario: SignalR down
- **WHEN** the manager does not consider the charger connected
- **THEN** connectivity is reported down and the reason is logged at debug level

#### Scenario: HTTP health check fails
- **WHEN** the Easee health endpoint fails
- **THEN** the ping result is failed

### Requirement: Diagnostic Error Report
The application SHALL install a single global logrus error hook, however many application instances
are constructed, and SHALL answer the diagnostic report request from it. Installing it more than once
would record every warning into the report twice.

#### Scenario: diagnostic report requested
- **WHEN** the app diagnostic report is requested
- **THEN** the warnings and errors captured by the hook are returned

#### Scenario: second application constructed
- **WHEN** a second application instance is built in the same process
- **THEN** the hook is not attached again
