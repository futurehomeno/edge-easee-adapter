# Tasks

## 1. An unconfirmed start succeeds
- [x] 1.1 Write a failing test: a start whose echo misses `current_wait_duration` returns no error
- [x] 1.2 Log the miss at info instead of failing in `StartChargepointCharging`, run the test green

## 2. Gate
- [x] 2.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 2.2 `make test`
- [x] 2.3 `openspec validate start-resume-wait --strict`
