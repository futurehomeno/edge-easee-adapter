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

## 3. A session end resets the session values (issue 43 point 3)
- [x] 3.1 Audit of the values the adapter keeps per session (design.md)
- [x] 3.2 Failing tests: `TestController_EndSession_ForgetsTheAdapterPause`,
      `TestController_EndSession_UnpluggedDropsAWaitingWrite`, `TestController_EndSession_UnpluggedDropsAWaitingPhaseBounce`,
      the session-end calls in `TestObservationsHandler_ChargerStateTellsWhetherTheSessionCharged`
- [x] 3.3 `EndSession` on the controller, called from the handler's session-finished branch
- [x] 3.4 Issue 43 decided and archived
- [x] 3.5 Review of PR 240: failing tests `TestController_EndSession_KeepsTheClearAgainstAWriteInFlight`,
      `TestController_EndSession_ForgetsTheSlowStart`, the call order in
      `TestObservationsHandler_SessionFinishedClearsRequestedCurrentWithNow`; a write stamps the cache with the time it
      left, `EndSession` runs before the clear and forgets the slow start
- [x] 3.6 Review of PR 240: failing test `TestController_EndSession_BeatsAFailedCommand`; a pause or resume failing
      across a session end does not put the forgotten pause back
- [x] 3.7 Issue 44 decided: failing test `TestController_UserStop_SubMinimumValueStartsAtTheInitialCurrent`; a kept
      value below the minimum starts like the adapter's own pause. Issue 45 decided: no change
- [x] 3.8 Review of PR 240: failing test `TestController_EndSession_BeatsASlowStartBeingSent`; a start being sent
      across a session end does not put the slow-start memory back

## 4. Docs and release
- [x] 4.1 README: the start current
- [x] 4.2 Version 3.4.3
