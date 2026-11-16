# Keep a finished session when its start is replayed

## Why
- On every signalR (re)connect Easee replays the latest session observations, stop first, then start
  with the same ID: `Stop session {244 55.2 kWh …}` then `Start session {244 …}`, seen on every
  reconnect of 2.6.7 and 3.1.4 hubs.
- `RegisterSessionStart` writes `{ID, Start}` under the ID unconditionally, so the replayed start
  overwrites the stored stop: the finished session loses its stop time and energy and reads as
  open while the charger is disconnected. The next real session then closes it with energy 0.

## What Changes
- Registering a session start whose ID is already stored leaves the stored session untouched.

## Impact
- Affected specs: `reporting`
- Affected code: `src/internal/db/db.go`
