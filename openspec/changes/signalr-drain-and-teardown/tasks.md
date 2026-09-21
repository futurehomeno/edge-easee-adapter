# Tasks

## 1. Connect deadline (#168) - no change needed

- [x] 1.1 Reproduce the Timer.Stop race on this module: it does not occur under `go 1.26.0`
- [x] 1.2 Measure both semantics (0/100000 vs 99972/100000) and record the finding on the issue

## 2. Queue alarm and inclusion reports (#167)

- [x] 2.1 Failing test: an alarm report does not block the observation drain
- [x] 2.2 Failing test: an inclusion report does not block the observation drain
- [x] 2.3 Enqueue both through the existing report queue
- [x] 2.4 Tests green

## 3. Stop the energy goroutine on Close (#166)

- [x] 3.1 Failing test: no meter report is published after Close
- [x] 3.2 Close the energy channel and wait for the goroutine
- [x] 3.3 Tests green, Close still idempotent

## 4. Gate

- [x] 4.1 `go build ./...`, `go vet ./...`, `golangci-lint run`
- [x] 4.2 `make test`
- [x] 4.3 `openspec validate signalr-drain-and-teardown --strict`
