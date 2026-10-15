# Topology observed guard

## Why
The newest-topology timestamp advanced only on a change. A newer observation equal to the cache moved
nothing, so an older, different one arriving after it still replaced the props and the cache.

## What Changes
- A grid-type observation older than the newest one observed, changed or not, is ignored.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/signalr/handler.go`
