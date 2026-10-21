# Tasks

## 1. Grid-aware translation
- [x] 1.1 Write a failing test: `output_phase=10` on an IT grid maps to `L1L2`, `11` on a TN grid to `NL1`, and an unknown grid keeps the suffix mapping
- [x] 1.2 Make `OutputPhaseType.ToFimpState` take the FIMP grid type and swap to the twin on a contradicting suffix, run the test green

## 2. Handler
- [x] 2.1 Write a failing handler test: an IT 1-phase charger reporting `P1T2T3TN` caches and stores `L1L2`
- [x] 2.2 Pass the cached grid type from `handleOutPhase`, run the test green

## 3. Gate
- [x] 3.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 3.2 `openspec validate output-phase-follows-grid-type --strict`
