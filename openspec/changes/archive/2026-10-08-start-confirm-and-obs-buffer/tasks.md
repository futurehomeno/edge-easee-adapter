# Tasks

## 1. Deferred write echo
- [x] 1.1 Write failing tests: the deferred send waits `offered_current_wait_time` for the echo, and Teardown does not wait for the echo
- [x] 1.2 Wait the window in `sendPending` and release `inFlight` once the send returns; run the tests green

## 2. Observation buffer
- [x] 2.1 Write failing tests: an unsupported observation takes no buffer slot, and a 20-charger replay fits undrained
- [x] 2.2 Filter in `receiver.ProductUpdate`, raise the buffer to 500, drop the manager's filter; run the tests green

## 3. Gate
- [x] 3.1 Bump to 3.2.2
- [x] 3.2 `go vet ./...`, `go build ./...`, `golangci-lint run`, `make lint`
- [x] 3.3 `make test`
- [x] 3.4 `openspec validate start-confirm-and-obs-buffer --strict`
