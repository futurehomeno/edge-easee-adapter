# Topology pending guard

## Why
The timestamp guard compared only with the cached grid type, but the cache is written after the queued
republish lands. An older topology arriving while a newer one was still queued replaced the props, and
the cache then kept the newer one, so the equality check skipped every later correct observation.

## What Changes
- A grid-type observation older than the newest accepted one, cached or still queued, is ignored.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/signalr/handler.go`
