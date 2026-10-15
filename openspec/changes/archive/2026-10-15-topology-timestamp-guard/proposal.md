# Topology timestamp guard

## Why
A grid-type observation older than the cached one replaced the props while the cache rejected it,
and the equality check against the cache then skipped every later correct observation until restart.

## What Changes
- A grid-type observation older than the cached grid type leaves the props and cache untouched.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/signalr/handler.go`
