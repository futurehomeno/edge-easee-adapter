# Re-sync the charger after a phase mode is sent

## Why
- Since `phase-set-before-echo` (3.3.0) the sent target replaces the cached internal phase mode.
  Easee can accept the command (200/202) while the charger never applies it. The value does not
  change, so no observation follows, and every repeat of the request equals the cached target and
  is skipped. Up to 3.2.5 the repeat was sent again (issue 10, v3.3.0 review).
- A replay carries the charger's real mode at the timestamp the cached target was stamped with, so
  it replaces the target. A charging charger gets one from the re-sync its session bounce arms; an
  idle charger gets none until something else reconnects.

## What Changes
- A sent phase mode arms the same 15 s re-sync a start or a stop does, also on an idle charger.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/easee/controller.go`
- One more `SubscribeWithCurrentState` invocation per phase-mode change of an idle charger. On a
  connection whose token has lapsed it is refused and reconnects the client, at most once per
  connection (issue 14), as the re-sync after a start or a stop already does.
