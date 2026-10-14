# Tasks
- [x] 1 Failing tests: fresh store enabled, old seed migrated once, version 8 leaves it alone
- [x] 2 Remove `seedTelemetryDisabled`, add the 7->8 migration, run green
- [x] 3 `go vet`, `go build`, `make lint`, `go test -short`, `openspec validate telemetry-default-on --strict`
