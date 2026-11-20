# Tasks

## 1. State report on power readings
- [x] 1.1 Write a failing test: `charging` reported, then `await_start` with power above zero (no
  report), then 0 W publishes `evt.state.report` `ready_to_charge`
- [x] 1.2 Publish an unforced state report from `handleTotalPower`, run the test green

## 2. Gate
- [x] 2.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 2.2 `make test`
- [x] 2.3 `openspec validate report-state-on-power --strict`
