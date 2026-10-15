# start-confirm-and-obs-buffer

## Why

The 3.1.4 beta logs (2026-10-08) show two defects that #182 (3.2.1) did not cover.

- **Deferred-write warnings on writes that landed.** `sendPending` warns
  `Deferred set_current N was not echoed back by the charger` after `current_wait_duration` (3s).
  Every such echo arrived 1–3s later. "Update dynamic current" → "Offered current" exceeded 3s
  for 21 of 172 writes on one hub (max 11s) and 8 of 168 on another (max 15s). Resuming from
  `await_start` is the slow case. The wait also counted toward Teardown's in-flight wait, so a longer
  bound would have held back `cmd.thing.delete`.
- **Lost session replays.** On every (re)connect Easee replays the full snapshot for each charger:
  over 100 observations, most of them IDs the adapter ignores. They were only filtered in the run
  loop, after they had taken a slot in the 100-slot buffer. Meanwhile the run loop writes sessions
  to buntdb with `SyncPolicy Always`. The receiver drops on a full buffer
  (`observation buffer full, dropping obs='unknown=149' chargerID=EH248211`). On hub 59e47463 the
  `Start session` replay was lost both times the adapter started.

`cmd.charge.start` stays as #182 left it: an unconfirmed start succeeds and is logged at info, and
the wait under the service lock keeps its 3s bound.

## What Changes

- A deferred write waits up to `offered_current_wait_time` (default 30s) for its echo instead of
  `current_wait_duration`. The next send cannot leave before then anyway. No new setting is needed.
- Teardown waits only for a deferred send to reach Easee, not for its echo.
- The SignalR receiver drops unsupported observation IDs before buffering. The buffer grows from
  100 to 500, and the run loop's now-redundant filter is removed.

## Impact

- Affected specs: `charging-control`, `observation-streaming`
- Affected code: `src/internal/easee/controller.go`, `src/internal/signalr/{receiver,client,manager}.go`
