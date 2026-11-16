## 1. Keep every leg advertised

- [x] 1.1 Failing test: an IT 3-phase charger moving from `L1L2` to `L3L1` keeps all four modes advertised and sends no inclusion report.
- [x] 1.2 Advertise `SettablePhaseModes`; the output phase handler only persists the leg.
- [x] 1.3 Update the tests that asserted the single-leg list; run green.

## 2. Review fixes

- [x] 2.1 Failing test: a store write that blocks or fails neither parks the observation drain nor drops the phase-mode report.
- [x] 2.2 Write the leg inside the queued phase-mode job.
- [x] 2.3 Spec: the unset-mode fallback of Phase Mode Report Precedence no longer names the persisted phase.

## 3. Verify

- [x] 3.1 `go build ./...`, `go vet ./...`, `golangci-lint run`.
- [x] 3.2 `go test ./internal/model ./internal/signalr ./internal/easee ./cmd/...`.
- [x] 3.3 `openspec validate advertise-every-leg --strict`.
