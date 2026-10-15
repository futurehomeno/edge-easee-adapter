# Tasks

- [x] 1 Failing test: the auth-loss handler marks the lifecycle `LOST` with the reason before logging out
- [x] 2 Set `LOST` in the auth-loss handler; pass the lifecycle in from the factory
- [x] 3 `go build ./...`, `go vet ./...`, `golangci-lint run`, `go test -short ./...`
- [x] 4 `openspec validate auth-loss-telemetry --strict`
