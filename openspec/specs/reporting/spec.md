# Reporting Specification

## Purpose
Publish charger telemetry to FIMP: operating state, electricity meter values, alarms, and charging
session history. Reports are driven by SignalR observations as they arrive, and by a periodic polling
task as a backstop.
## Requirements
### Requirement: Charger State Report
`evt.chargepoint.state_report` SHALL report the cached charger state, except that a cached total
power greater than zero SHALL be reported as `charging` when its observation is not older than the
cached charger-state observation. A charger-state observation newer than the last power reading
SHALL win, so a power value the stream stopped updating cannot hold the charger at `charging`. The
chargepoint service SHALL advertise in `sup_states` the FIMP states `unknown`, `disconnected`,
`ready_to_charge`, `charging`, `finished`, `error`, `suspended_by_ev` and `requesting` — Easee's nine
operating states mapped to FIMP and deduplicated, `offline` and `de-authenticating` both collapsing
to `unknown`. The FIMP names are not the Easee ones: Easee's awaiting start maps to
`ready_to_charge`, while Easee's ready to charge maps to `suspended_by_ev`.

#### Scenario: power observed while state says otherwise
- **WHEN** the cached total power is above zero and its observation is not older than the cached state
- **THEN** `charging` is reported

#### Scenario: newer state after the last power reading
- **WHEN** the cached total power is above zero but a non-charging state was observed after it
- **THEN** the cached charger state is reported

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
an unforced extended report for `power_import`, and then an unforced chargepoint state report, so a
state change that the power reading reveals is published without waiting for the next state
observation. The state report SHALL be attempted even when a meter report fails, and SHALL NOT be
sent until a charger state has been observed.

#### Scenario: power observation arrives
- **WHEN** a total-power observation of 7.4 arrives
- **THEN** 7400 is cached and the two unforced meter reports are published

#### Scenario: power drops to zero after a state masked by residual power
- **WHEN** `charging` was the last state report, an `await_start` state observation arrived while a
  power reading above zero not older than it was cached, and then a 0 W power observation arrives
- **THEN** one `evt.state.report` with `ready_to_charge` is published after the 0 W reading

#### Scenario: power reading without a state change
- **WHEN** a power observation arrives and the reported state is unchanged
- **THEN** no state report is published

#### Scenario: meter report fails
- **WHEN** `ready_to_charge` is cached, a 0 W power observation arrives and the `W` meter report
  fails
- **THEN** the unforced state report is still published and the meter report error is logged

#### Scenario: power reading before any charger state
- **WHEN** no charger state has been observed and a 0 W power observation arrives
- **THEN** the two meter reports are published and no state report

### Requirement: Phase Current Observations
The in-current observations SHALL populate the phase 1, 2 and 3 currents from the terminals that
carry the lines on the grid type held in the chargepoint props: T3, T4 and T5 on a TN grid and while
the props carry no grid type, T2, T3 and T4 on an IT or TT grid. An in-current observation for a
terminal that carries no line on that grid (T2 on TN, T5 on IT or TT) SHALL be ignored: no cache
write, no report and no error. The handler SHALL read the grid type from the props once and keep it
until a grid-type observation replaces the props, so an ignored observation costs no service lookup.
Each accepted observation SHALL publish its own unforced extended meter report on a successful cache
update.

#### Scenario: phase current observed
- **WHEN** an in-current T3 observation arrives newer than the cached value on a TN charger
- **THEN** the phase 1 current is cached and an unforced extended report is published

#### Scenario: IT or TT charger
- **WHEN** a charger whose props carry grid type IT or TT reports in-current T2, T3 and T4
- **THEN** they are cached as the phase 1, 2 and 3 currents, each with its own extended report

#### Scenario: terminal without a line
- **WHEN** a TN charger reports in-current T2, or an IT or TT charger reports in-current T5, even
  with a value that does not parse
- **THEN** no phase current is written, no report is published and the handler returns no error

#### Scenario: repeated neutral current
- **WHEN** a TN charger reports in-current T2 twice
- **THEN** the props are read for the first reading only

#### Scenario: grid type unknown
- **WHEN** the props carry no grid type and in-current T2, T3, T4 and T5 arrive
- **THEN** T3, T4 and T5 are cached as the phase 1, 2 and 3 currents and T2 is ignored

#### Scenario: topology republish still pending
- **WHEN** the props already carry IT while the cache holds no or another grid type, and in-current
  T2 arrives
- **THEN** it is cached as the phase 1 current

#### Scenario: grid type changes after a reading
- **WHEN** in-current T2 was ignored under TN props, a grid-type observation then replaces them with
  IT, and in-current T2 arrives again
- **THEN** it is cached as the phase 1 current

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
time. Registering a session start whose ID is already stored SHALL leave that session unchanged.
Registering a session stop SHALL write the session's start, stop and energy.

#### Scenario: previous session left open
- **WHEN** a session start is registered and the latest stored session has a zero stop time
- **THEN** that session's stop is set to the new session's start before the new one is stored

#### Scenario: start replayed after its stop
- **WHEN** a session start is registered for a session ID that is already stored with a stop time
- **THEN** the stored session keeps its start, stop and energy

#### Scenario: start of an older stored session replayed
- **WHEN** a session start is registered for a stored session ID older than the latest session
- **THEN** both sessions stay unchanged and the latest one is not closed

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

### Requirement: Connectivity Reported On Change
The adapter SHALL send a charger's node report as soon as its connectivity changes, without waiting
for the periodic connectivity task: when its SignalR subscribe succeeds or fails, when the SignalR
client disconnects, and when the charger's cloud-connected or offline state flips. The report SHALL
be non-forced, so an unchanged status is deduped by the reporting cache.

#### Scenario: login after a logout
- **WHEN** the SignalR client reconnects after a login and the charger resubscribes
- **THEN** a node report with connectivity up is sent right away

#### Scenario: SignalR connection lost
- **WHEN** the SignalR client disconnects
- **THEN** a node report with connectivity down is sent for every registered charger

#### Scenario: charger leaves the cloud
- **WHEN** the charger's cloud-connected observation flips to false or back to true
- **THEN** a node report is sent with the new connectivity

