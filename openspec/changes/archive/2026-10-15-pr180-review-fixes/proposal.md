# PR #180 review fixes

## Why
- A failed re-sync invoke cleared the re-sync request first, so its armed retry found the charger
  subscribed and did nothing: the replay was lost.
- The output phase mapped through the cached grid type, which a topology republish still queued has
  not written yet, while the advertised modes are built from the props it already replaced.

## What Changes
- A failed re-sync keeps the request, so the retry re-invokes the subscribe.
- The output phase maps through the grid type in the chargepoint props.

## Impact
- Affected specs: `observation-streaming`, `phase-mode`
- Affected code: `src/internal/signalr/manager.go`, `src/internal/signalr/handler.go`
