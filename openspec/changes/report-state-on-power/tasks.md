# Tasks

## 1. State report on power readings
- [x] 1.1 Write a failing test: `charging` reported, then `await_start` with power above zero (no
  report), then 0 W publishes `evt.state.report` `ready_to_charge`
- [x] 1.2 Publish an unforced state report from `handleTotalPower`, run the test green
- [x] 1.3 Write a failing test: a failed `W` meter report still sends the state report and logs
  the error; attempt all three reports and join their errors
- [x] 1.4 Write a failing test: a 0 W reading before any charger state sends no state report; send
  it only once the cached charger state has a timestamp
- [x] 1.5 Bump cliffhanger v1.5.0 -> v1.5.3 (#287: forced state report before a start/stop error)

## 2. Gate
- [x] 2.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 2.2 `make test`
- [x] 2.3 `openspec validate report-state-on-power --strict`
