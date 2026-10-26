## 1. Keep every leg advertised

- [x] 1.1 Failing test: an IT 3-phase charger moving from `L1L2` to `L3L1` keeps all four modes advertised and sends no inclusion report.
- [x] 1.2 Advertise `SettablePhaseModes`; the output phase handler only persists the leg.
- [x] 1.3 Update the tests that asserted the single-leg list; run green.

## 2. Verify

- [x] 2.1 `go build ./...`, `go vet ./...`, `golangci-lint run`.
- [x] 2.2 `go test ./internal/model ./internal/signalr ./internal/easee ./cmd/...`.
- [x] 2.3 `openspec validate advertise-every-leg --strict`.
