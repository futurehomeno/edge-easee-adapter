# Tasks

## 1. A stored stop holds the slot (#162)

- [x] 1.1 Failing test: a stored stop is not displaced by a later different-current write
- [x] 1.2 Failing test: a stop still replaces a stored current write
- [x] 1.5 Rewrite `StopChargepointCharging_IsReplacedByAStart`: an explicit start no longer cancels
      a stored stop either (user decision, 2026-09-21)
- [x] 1.6 Pin the phase-mode bounce whose pause is itself deferred
- [x] 1.3 Drop a current write that arrives while a stop holds the slot
- [x] 1.4 Tests green

## 5. Reverse 1: the last command always wins (user decision, 2026-09-21)

Section 1 left a phase-mode change able to stop charging for good: the restart's resume was dropped
against its own pause. The slot policy is reverted and a restart becomes an ordered pair.

- [x] 5.1 Failing test: a phase-mode restart sends the pause and then the resume, leaving the
      charger charging
- [x] 5.2 Failing test: `start stop start stop` and `set_current stop set_current stop` inside one
      window each send exactly the last command
- [x] 5.3 Failing test: a write ending the stream displaces a stored stop (both same and different
      value, covering the dedup path)
- [x] 5.4 Failing test: a later command discards both halves of a stored restart; teardown likewise
- [x] 5.5 Invert `pendingDiffers` so a stored stop always counts as differing
- [x] 5.6 Drop the stop-holds-the-slot guard from `dispatch`; log `<new> preempts <old>` on a
      displacement
- [x] 5.7 Add the follow-up slot and dispatch the restart as an ordered pair
- [x] 5.8 Clear the follow-up in `Teardown`
- [x] 5.9 Tests green
- [x] 5.10 Deadline does not move when a command displaces another (10 rapid writes, one send)
- [x] 5.11 Bump VERSION to 3.1.4

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
