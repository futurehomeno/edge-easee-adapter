# Tasks

## 1. Report with internal mode 0
- [x] 1.1 Write failing controller tests: TN 1-phase and TN 3-phase charger config with `PhaseMode` 0 and an empty output-phase cache report an advertised mode without error
- [x] 1.2 Fall back to the advertised modes in `ChargepointPhaseModeReport` when the internal mode is 0, run the tests green

## 2. Gate
- [x] 2.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 2.2 `make test`
- [x] 2.3 `openspec validate report-phase-mode-when-unset --strict`
