# edge-easee-adapter

A Futurehome hub edge adapter that exposes Easee EV chargers as FIMP devices. It talks to the
Easee cloud over HTTPS for commands and configuration, and consumes live charger telemetry over
a SignalR websocket.

## Tech stack

- Go, module rooted at `src/` (`github.com/futurehomeno/edge-easee-adapter`)
- `github.com/futurehomeno/cliffhanger` — adapter framework: things, services, lifecycle, routing,
  tasks, config service, auth, storage
- `github.com/futurehomeno/fimpgo` — FIMP message transport and type definitions
- `github.com/philippseith/signalr` — SignalR client for the observation stream
- `buntdb` via `cliffhanger/database` — charging session history
- `mockery` for test doubles, `golangci-lint` for static analysis

## Layout

| Path | Responsibility |
|---|---|
| `src/cmd/` | process wiring: service graph, root builder, factories |
| `src/internal/app/` | app lifecycle: login, logout, configure, uninstall, charger seeding |
| `src/internal/api/` | Easee HTTP client, authenticator, token exchange |
| `src/internal/signalr/` | SignalR client, subscription manager, observation handlers |
| `src/internal/easee/` | thing factory, chargepoint controller, connector |
| `src/internal/cache/` | per-charger timestamped observation cache |
| `src/internal/model/` | Easee wire types, observation IDs, phase-mode matrix |
| `src/internal/config/` | settings, migrations, credential store |
| `src/internal/db/` | charging session storage |
| `src/internal/routing/`, `src/internal/tasks/` | FIMP routes and background tasks |

## Conventions

- Log lines are mostly prefixed with a bracketed component (`[app]`, `[auth]`, `[db]`) or the
  charger ID; older lines in `routing/`, `signalr/` and `tasks/` are still unprefixed.
- Errors are wrapped with `fmt.Errorf("...: %w", err)` and logged once at the boundary;
  `api/http.go` and `db/db.go` still wrap with `github.com/pkg/errors`.
- Every value the cache holds carries a timestamp: for observations the time the value was observed,
  for controller-requested values (`requestedOfferedCurrent`, `requestedPhaseMode`) the time the
  command was issued. A write with an older timestamp than the value already stored is rejected.

## Baseline note

These specs document the behaviour as implemented at the time they were written. Behaviour that
is tracked as a defect (issues #107-#112) is recorded here as it currently is, not as it should
be.
