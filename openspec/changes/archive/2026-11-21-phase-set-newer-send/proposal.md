# Send a phase mode the cache holds when another one was sent since

## Why
- Issue 11: auto and then single phase are sent. The auto echo is newer than the observation the
  single-phase target replaced, so the cache goes back to auto. A three-phase (auto) request made
  before the single-phase echo lands equals the cache and is skipped as a no-op; the single-phase
  echo then lands and the charger stays in single phase although auto was the last request.
- Issue 10: a replay landing before the echo (a SignalR reconnect, a `Resync`, the stale watchdog,
  or the pending re-sync a phase send keeps) puts the pre-send mode back, so a request for that mode
  is skipped while the charger may still apply the send.
- Field data (issue 10): the internal-mode echo arrives 1-2 s after a send, well inside the 15 s
  re-sync window.

## What Changes
- The controller remembers the last phase target Easee accepted and when (hub clock). A target equal
  to the cached internal mode is still skipped, unless a different target was sent less than 15 s
  before: then it is sent.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/easee/controller.go`
- A mode changed elsewhere (the Easee app) within 15 s of a send, and then requested, is sent again:
  one more Easee call, and one more bounce of a charging session.
