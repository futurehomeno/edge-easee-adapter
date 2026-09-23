## ADDED Requirements

### Requirement: State Re-sync After Start Or Stop
After `cmd.charge.start` or `cmd.charge.stop` is sent to Easee and accepted, the adapter SHALL request a state re-sync for that charger 15 s after the send, because Easee's live stream may never deliver the state, power or phase-current change the command causes. At most one re-sync per charger SHALL be pending; a later accepted start or stop SHALL replace it with one 15 s after the later send. A command that fails, or that is stored for a later send, SHALL NOT schedule a re-sync until it is actually sent.

#### Scenario: stop accepted
- **WHEN** a pause-charging command is sent and Easee accepts it
- **THEN** a state re-sync for that charger is requested 15 s later

#### Scenario: start accepted
- **WHEN** a start or resume command is sent and Easee accepts it
- **THEN** a state re-sync for that charger is requested 15 s later

#### Scenario: second command within the delay
- **WHEN** another start or stop is accepted before the pending re-sync runs
- **THEN** only one re-sync runs, 15 s after the later send

#### Scenario: command fails
- **WHEN** the start or stop request fails
- **THEN** no re-sync is scheduled

#### Scenario: stop stored by the pacing window
- **WHEN** a stop is stored because a write went out within `offered_current_wait_time`
- **THEN** the re-sync is scheduled 15 s after the stored stop is actually sent

### Requirement: Charging Watchdog
The adapter SHALL check every minute each charger whose cached state is `charging`. When no `TotalPower`, `InCurrentT3`, `InCurrentT4` or `InCurrentT5` observation has arrived for that charger for 5 minutes, it SHALL request a state re-sync, and while the silence lasts it SHALL request at most one re-sync every 5 minutes. `DynamicChargerCurrent` observations SHALL NOT count as activity, because Easee echoes the adapter's own writes with them while the rest of the stream is silent.

#### Scenario: charging and silent
- **WHEN** a charger is cached as `charging` and no power or phase-current observation arrived for 5 minutes
- **THEN** a state re-sync is requested for it

#### Scenario: only command echoes arrive
- **WHEN** the only observations for a charger cached as `charging` are `DynamicChargerCurrent`
- **THEN** the watchdog still requests a re-sync after 5 minutes

#### Scenario: not charging
- **WHEN** a charger's cached state is anything other than `charging`
- **THEN** the watchdog requests no re-sync however long the stream is silent

#### Scenario: silence continues
- **WHEN** the silence continues after a watchdog re-sync
- **THEN** the next re-sync comes no sooner than 5 minutes after the previous one
