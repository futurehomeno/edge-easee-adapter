# Design

## Decision
Write the sent target into the cached internal phase mode, stamped with the timestamp of the
observation it replaces (`internalAt`, read before the call).

- The timestamps `requestStillHolds`, `outputPhaseStale` and `recordAt` compare are unchanged, so the
  report precedence and the request stamping behave as they did between a send and its echo.
- The cache rejects only an older timestamp, so every newer phase-mode observation replaces the sent
  target, including one stamped behind the hub clock: the charger moving elsewhere, or refusing the
  mode, is still seen. An observation that landed during the HTTP call keeps the cache.
- The echo of the sent mode now equals the cached value and is ignored, as `Phase Mode Observation
  Handling` already says for a repeat; the report it used to queue was unforced, and the requested
  mode it would have found is the one `cmd.phase_mode.set` already reported.

## Rejected
- A fresh stamp (hub clock, or `recordAt`): it can outrank a newer observation from a clock behind
  it, keeping a mode the charger never took and skipping the next request for it.
- Comparing against a last-sent target held in the controller: a second source of the internal mode
  that needs its own timestamp rule and lock, for the same comparison the cache already serves.
