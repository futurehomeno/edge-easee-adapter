# A sent phase mode arms its re-sync only where it is needed

## Why
Review of PR #219 (`phase-set-resync`), which armed the re-sync on every sent phase mode:
- For a charging charger whose bounce stop is stored by the pacing window, the re-sync ran before
  that stop was sent. A replay that beat the echo put the old mode back in the cache, so a request
  for the old mode was dropped while the stored bounce still applied the new one. On 3.3.0 that
  request was sent.
- A phase set replaced the pending re-sync, so it pushed back the one a start or a stop armed.
  Phase sets are not paced, so a run of them could hold it back past energy guard's 60 s wait.
- A phase set still waiting for Easee during Teardown armed a re-sync after Teardown had cancelled
  it.

## What Changes
- A sent phase mode arms the re-sync only when no session is restarted. A restart is re-synced by
  its own stop and resume, as on 3.3.0.
- A phase mode keeps a pending re-sync instead of replacing it.
- No re-sync is armed once Teardown has run.

## Impact
- Affected specs: `phase-mode`, `charging-control`
- Affected code: `src/internal/easee/controller.go`
- A phase mode sent while a re-sync is pending is replayed by that one, less than 15 s after the
  send.
