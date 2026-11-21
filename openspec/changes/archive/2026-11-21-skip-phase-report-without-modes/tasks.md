# Tasks

## 1. Report gate
- [x] 1.1 Write a failing test: a charger without `sup_phase_modes` gets an internal-mode change and
  an output-phase change; no `aware phase modes is not supported` warning is logged and the leg is
  persisted
- [x] 1.2 Skip the queued phase-mode report when the chargepoint is not phase-mode aware, still
  persisting the leg; run the test green

## 2. Existing tests
- [x] 2.1 The mocked chargepoint services of the existing output-phase tests answer
  `IsPhaseModeAware` with true; what they assert is unchanged

## 3. Gate
- [x] 3.1 `go test ./internal/signalr/`, also with `-race`
- [x] 3.2 `go build ./...`, `go vet ./...`, `golangci-lint run ./...`
- [x] 3.3 `openspec validate skip-phase-report-without-modes --strict`
