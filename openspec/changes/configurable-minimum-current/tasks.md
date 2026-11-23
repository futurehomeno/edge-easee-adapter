# Tasks

## 1. The settings
- [x] 1.1 Failing tests: `TestService_SetMinChargingCurrent`, `TestService_InitialChargingCurrentNotBelowMinimum`,
      `TestService_RaisingMinimumRaisesInitialChargingCurrent`, the packaged default
- [x] 1.2 `min_charging_current` and the `initial_charging_current` setter, routes, packaged `config.json`

## 2. Below the minimum pauses
- [x] 2.1 Failing tests: `TestController_SetChargepointOfferedCurrent_BelowMinimumPausesAndNextValueResumes`,
      `TestController_SetChargepointOfferedCurrent_UserStopIsNotResumed`
- [x] 2.2 Failing tests: `TestDivertOfferedPause_UnderSixAmpsReachesTheControllerAndRenames`,
      `TestDivertOfferedPause_LeavesOtherValuesToTheFramework`, `TestDivertOfferedPause_FailedPauseKeepsTheMessage`
- [x] 2.3 Controller, thing factory lookup and routing

## 2b. PR 235 review
- [x] 2b.1 Failing tests: `TestController_SetChargepointOfferedCurrent_PauseDuringWriteIsStillResumed`,
      `TestService_ConcurrentMinimumAndInitialKeepInitialNotBelowMinimum`
- [x] 2b.2 The pause mark is taken before the write; both setters compare inside the config update

## 2c. A user stop holds until a start (issue 33)
- [x] 2c.1 Spec: A User Stop Holds Until A Start; MODIFIED Stop Charging and The Last Command Always Wins
- [x] 2c.2 Failing tests: db round-trip, controller (value, sub-minimum, start, failed start, restart, phase bounce),
      handler (unplugged, charging after the stop, older charging observation, offline); the four displacement tests
      changed to the new rule
- [x] 2c.3 `SetUserStop`/`UserStop` in the session storage, the controller and the handler

## 2d. PR 235 review, user stop races
- [x] 2d.1 Failing tests: `TestEndUserStop_KeepsAStopMadeMeanwhile`,
      `TestController_SetChargepointOfferedCurrent_StopAfterTheCheckHolds`
- [x] 2d.2 `EndUserStop` reads and clears under the storage lock; every write re-checks the user stop under `pace`

## 3. Gate
- [x] 3.1 `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`
- [x] 3.2 `openspec validate configurable-minimum-current --strict`
