# Tasks

## 1. Name the internal modes
- [x] 1.1 Write a failing test for a `PhaseModeName` mapping 1/2/3 to `single phase`/`auto`/`three phase` and anything else to `unknown`
- [x] 1.2 Add the `easeePhaseModeThree` constant and `PhaseModeName` to `model/mapper.go`, run the test green

## 2. Log the name
- [x] 2.1 Write a failing test asserting `SetPhaseMode` logs `Set phase mode to auto (2)`
- [x] 2.2 Change the format string in `api/client.go` to use the name, run the test green

## 3. Gate
- [x] 3.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 3.2 `make test`
- [x] 3.3 `openspec validate log-phase-mode-name --strict`
