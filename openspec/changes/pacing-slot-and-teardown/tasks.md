# Tasks

## 1. A stored stop holds the slot (#162)

- [x] 1.1 Failing test: a stored stop is not displaced by a later different-current write
- [x] 1.2 Failing test: a stop still replaces a stored current write
- [x] 1.5 Rewrite `StopChargepointCharging_IsReplacedByAStart`: an explicit start no longer cancels
      a stored stop either (user decision, 2026-09-21)
- [x] 1.6 Pin the phase-mode bounce whose pause is itself deferred
- [x] 1.3 Drop a current write that arrives while a stop holds the slot
- [x] 1.4 Tests green

## 2. Cancel the deferred timer on teardown (#161)

- [x] 2.1 Failing test: a stored command is not sent after the connector disconnects
- [x] 2.2 Keep the timer handle in `dispatch`; add controller teardown that stops it and clears the slot
- [x] 2.3 Call teardown from `connector.Disconnect`
- [x] 2.4 Tests green

## 3. Start survives an unseeded cache (#163)

- [x] 3.1 Failing test: start with an empty cache starts at 32A instead of failing
- [x] 3.2 Seed the cache's max current from the persisted supported max current in the factory
- [x] 3.3 Fall back to the 32A clamp in the start path
- [x] 3.4 Tests green

## 4. Gate

- [x] 4.1 `go build ./...`, `go vet ./...`, `golangci-lint run`
- [x] 4.2 `make test`
- [x] 4.3 `openspec validate pacing-slot-and-teardown --strict`
