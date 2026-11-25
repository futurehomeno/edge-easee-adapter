# edge-easee-adapter

Go edge adapter exposing Easee EV chargers as FIMP devices on the Futurehome hub (cliffhanger, fimpgo).
Commands go over the Easee HTTPS API, live telemetry over SignalR; layout in `openspec/project.md`.

## Build and test

- `make test`: all tests with `-race` (run `make generate-mocks` first; some tests start an MQTT
  test container)
- `make build-local`: build the binary
- Lint, in `src/`: `go vet ./... && go build ./...` and `golangci-lint run` (`src/.golangci.yaml`)

## Spec-driven development (OpenSpec)

This project is developed spec-first with OpenSpec. `openspec/specs/` describes what the code
does; it and the code must agree.

- Every change starts in `openspec/changes/<id>/`: `proposal.md`, `tasks.md`, `design.md` when a
  decision needs recording, and spec deltas in `specs/<capability>/spec.md` (ADDED / MODIFIED /
  REMOVED requirements with WHEN/THEN scenarios).
- `openspec validate <id> --strict` must pass before the code is written.
- When the change is merged, archive it (`openspec archive <id>`), so the deltas are folded
  into `openspec/specs/`. A finished change does not stay in `openspec/changes/`.
- Problems, doubts and open questions go to `openspec/issues/<NUM>-<name>.md` (next free number
  over open and archived issues); resolved ones move to `openspec/issues/archive/`.
- Vendor documents go to `openspec/docs/` with a row in `openspec/docs/README.md`.

## Test-driven development

Code is written test-first: for each behaviour write the failing test, run it and see it fail
for the expected reason, then write the code, then run it green. No code before its test.
Do not change what an existing test asserts unless the new behaviour clearly requires it.
