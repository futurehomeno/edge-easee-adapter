# Tasks

## 1. Pending topology
- [x] 1.1 Write a failing test: grid type then output phase with the topology republish still queued advertises the reported leg
- [x] 1.2 Read the topology from the chargepoint props in `persistOutputPhase`, run the tests green

## 2. Gate
- [x] 2.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 2.2 `make test`
- [x] 2.3 `openspec validate output-phase-pending-topology --strict`
- [x] 2.4 `openspec archive output-phase-pending-topology --yes` in this PR
