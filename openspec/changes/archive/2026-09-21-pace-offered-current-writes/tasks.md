# Tasks — pace-offered-current-writes

Test-first. Each test is run red before its increment and its failure line recorded here; the
increment then turns it green. Batch 1 (T1–T5) is the channel; batch 2 (T6–T10) is "a deferral
is a success" for start and resume. T6 sits in batch 2 because a deferred start reporting success
is what it asserts.

- [x] 1. OpenSpec change written, `openspec validate pace-offered-current-writes --strict` green
- [x] T1. `config_test.go` `TestService_OfferedCurrentDeferralTime_MatchesPackagedDefault` — 45 s
      packaged and unset. Red: `config.NewService(st).OfferedCurrentDeferralTime undefined`.
      **Dropped in review**: the setting was folded into `offered_current_wait_time`, now 30 s
      (design.md, "One window, not two"). The channel tests pin a 20 s window through
      `pacedConfig()`, so the timings below hold regardless of the packaged default.
- [x] T1b. `TestConfig_MigrateOfferedCurrentWaitTime` lifts `15s` and `20s` to `30s`;
      `TestService_OfferedCurrentWaitTime_MatchesPackagedDefault` reads 30 s; the cmd migration
      tests expect config version 7. Red: `expected: 30s actual: 20s`, `expected: 7 actual: 6`.
      Green with the migration step 6->7 and the packaged default.
- [x] T2. `api/http_test.go` `TestClient_UpdateDynamicCurrent_IsNotThrottled` (replaces
      `_ForceBypassesTheLocalThrottle`) — two writes and a stop back-to-back all reach the server.
      Red: second write `client: failed to update dynamic current: too many requests`
- [x] T3. `controller_test.go` `TestController_OfferedCurrentChannel_PacesWrites` — log replay:
      10 at t0 sent; 11..13 every 5 s stored; +19 s still one call; +20 s one send with 13;
      +25 s 14 stored; +40 s 14 sent; three `UpdateDynamicCurrent` in total. Red: unexpected
      `UpdateDynamicCurrent("test-charger", 11, false)` - the +5 s write went out
- [x] T4. `…_SendsAfterTheWindow` — set at t0, +20 s a different value goes out synchronously
      (the `>=` boundary), +39 s stored, +40 s sent. Red: 3 calls where 2 were expected - the
      +39 s write went out
- [x] T5. `TestController_StopChargepointCharging_StampsTheChannel` — stop at t0 sent; +1 s set 10
      stored; +19 s sent. Red: `Should not have called with given arguments` -
      `UpdateDynamicCurrent(10)` went out 1 s after the stop
- [x] A. Implement: `chargerCommand`, `pace`/`sentAt`/`pending`, `dispatch`, `send`,
      `sendPending`; `setOfferedCurrent` clamps, dedups (`!force`, on `clock.Since`) then
      dispatches, a deferral still answering `(false, nil)`; `StopChargepointCharging` dispatches
      a stop; client throttle, `force` and `cfgSrv` removed from `api`; callers and
      mocks updated. T1–T5 green, existing tests green.
- [x] T6. `TestController_StopChargepointCharging_IsReplacedByAStart` — set 10 at t0; +5 s stop
      stored; +5 s start returns nil; +10 s `UpdateDynamicCurrent(16)` once, `StopCharging`
      never. Red: `start accepted, but the charger did not resume at 16A`
- [x] T7. `TestController_SetChargepointPhaseMode_DefersTheResume` (replaces
      `_RestartsActiveSession`; the defect reproduction) — returns nil; `StopCharging` called;
      `UpdateDynamicCurrent` not called synchronously; +20 s `UpdateDynamicCurrent(16)` once and
      `SetRequestedOfferedCurrent(16, t0+20s)`. Red: `phase mode set to 1, but the charger did not
      resume at 16A`
- [x] T8. `…_ResumesAtTheSessionCurrent` (folds `_ResumesAtTheEvseCurrentAfterRestart` and
      `_ResumesAtTheSessionCurrent` into a table) — deferred payload is 6, never 16 or 32. Red:
      `phase mode set to 1, but the charger did not resume at 6A` (both rows)
- [x] T9. `…_DeferredResumeOutcomesAreLogOnly` (replaces `_ReportsUnconfirmedResume` and
      `_ReportsFailedResume`) — refused and unconfirmed deferred resume both return nil. Red:
      `phase mode set to 1, but the charger did not resume at 16A` (both rows)
- [x] T10. `TestController_StartChargepointCharging_DeferredStartReportsSuccess` — set 10 at t0;
      +5 s start returns nil without a synchronous echo wait; +15 s `UpdateDynamicCurrent(16)`
      then the echo wait. Red: `start accepted, but the charger did not resume at 16A`
- [x] B. Implement: a deferral answers `(true, nil)` from `setOfferedCurrent`;
      `restartForPhaseMode` keeps the pause's warn-and-return-nil and its tail is
      `setOfferedCurrent(resume, true)` with only the error branch; stale comments rewritten.
      T6–T10 green.
- [x] 2. `make generate-mocks`; `go vet`, `go build`, `golangci-lint run`, `staticcheck` green
- [x] 3. `make test` green. Coverage (statements / functions): `internal/easee` 85.3% / 87.0%,
      `internal/api` 86.1% / 93.0%, `internal/config` 96.2% / 100%; the channel itself:
      `dispatch` and `send` 100%, `sendPending` 84.6%. The lift beyond the change came from
      three table tests over untested wrappers: `api/client_test.go`, the controller's report
      methods, and the config setters/getters.
- [x] 4. Draft PR against `release_3.1`; #151 commented and closed as superseded
