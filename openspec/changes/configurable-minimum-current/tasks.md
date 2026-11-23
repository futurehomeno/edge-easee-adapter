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

## 3. Gate
- [x] 3.1 `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`
- [x] 3.2 `openspec validate configurable-minimum-current --strict`
