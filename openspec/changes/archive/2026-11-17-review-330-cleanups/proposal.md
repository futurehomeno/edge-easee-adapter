# v3.3.0 review cleanups

## Why
The review of v3.3.0 (`main...release_3.3`) raised three items that are not bugs:
- `InCurrentT2` is supported for every charger since `it-phase-currents`. On TN it is the neutral
  current, which streams as often as the line currents, and `handlePhaseCurrent` looked the
  chargepoint service up for every reading before it could drop it.
- `handlePhaseCurrent` and `handleOutPhase` read the grid type from the chargepoint props with the
  same lookup, type assertion and comment.
- `manager.restartClient` was extracted for the unauthorized restart, which now goes through
  `Client.Restart()`. Its one caller left is `Unregister`, where it is not a restart.

## What Changes
- The handler reads the props' grid type through one helper, keeps it, and drops it when a grid-type
  observation replaces the props. An ignored in-current reading costs no service lookup.
- `restartClient` is inlined back into `Unregister`. No behaviour change.

## Impact
- Affected specs: `reporting`
- Affected code: `src/internal/signalr/handler.go`, `src/internal/signalr/manager.go`
