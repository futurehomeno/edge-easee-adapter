## MODIFIED Requirements

### Requirement: Charging Watchdog
The adapter SHALL check every minute each charger whose cached state is `charging`. When no `ChargerOPState` observation and no `TotalPower` observation above zero, the two the state report is built from, has arrived for that charger for 5 minutes, it SHALL request a state re-sync, and while the silence lasts it SHALL request at most one re-sync every 5 minutes. `DynamicChargerCurrent` observations SHALL NOT count as activity, because Easee echoes the adapter's own writes with them while the rest of the stream is silent.

#### Scenario: charging and silent
- **WHEN** a charger is cached as `charging` and no state or power observation arrived for 5 minutes
- **THEN** a state re-sync is requested for it

#### Scenario: only command echoes arrive
- **WHEN** the only observations for a charger cached as `charging` are `DynamicChargerCurrent`
- **THEN** the watchdog still requests a re-sync after 5 minutes

#### Scenario: only zero power arrives
- **WHEN** the only observations for a charger cached as `charging` are `TotalPower` of zero
- **THEN** the watchdog still requests a re-sync after 5 minutes

#### Scenario: not charging
- **WHEN** a charger's cached state is anything other than `charging`
- **THEN** the watchdog requests no re-sync however long the stream is silent

#### Scenario: silence continues
- **WHEN** the silence continues after a watchdog re-sync
- **THEN** the next re-sync comes no sooner than 5 minutes after the previous one
