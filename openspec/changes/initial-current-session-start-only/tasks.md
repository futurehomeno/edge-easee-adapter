# Tasks

## 1. The session-charged value
- [x] 1.1 Failing test `TestObservationsHandler_ChargerStateTellsWhetherTheSessionCharged`: a charging observation
      sets it, the session-finished observation clears it, a pause keeps it
- [x] 1.2 `SessionCharged`/`SetSessionCharged` in the cache, set and cleared in `handleChargerState`

## 2. Session start and resume
- [x] 2.1 Failing tests: `TestController_StartChargepointCharging_ResumeAfterAPauseKeepsTheBalancedCurrent` (issue 42),
      `TestController_StartChargepointCharging_AfterTheAdapterPauseStartsAtTheInitialCurrent`, the resume rows of
      `TestController_StartChargepointCharging`; the session-start rows (8 A or 7 A cached: 16 A, nothing cached: the
      charger maximum) hold unchanged
- [x] 2.2 `StartChargepointCharging`
- [x] 2.3 The existing start tests: the start floor applies to a session start only

## 3. Docs and release
- [x] 3.1 README: the start current
- [x] 3.2 Version 3.4.3
