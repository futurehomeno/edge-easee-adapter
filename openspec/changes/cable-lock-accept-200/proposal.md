# Accept 200 OK on the cable lock command

## Why
Easee answers `POST /api/chargers/{id}/commands/lock_state` with `200 OK` and the queued command
(seen on a 3.1.4 hub). The client only accepted `202`, so a lock that went through failed
`cmd.param.set cable_always_locked` and skipped the cache seed.

## What Changes
- The cable lock command accepts `200` and `202`, like the phase mode command.

## Impact
- Affected specs: `charging-control`
- Affected code: `src/internal/api/http.go`
