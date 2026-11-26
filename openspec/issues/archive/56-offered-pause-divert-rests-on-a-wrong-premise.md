# 56 The offered-current divert rests on a wrong premise

## What happened
`routeOfferedPause` (`src/internal/routing/router.go`) hands `cmd.current_session.set_current`
values from 0 to 5 A to the controller before cliffhanger's handler, because the code and spec said
cliffhanger rejects an offered current under 6 A. It does not: `chargepoint/service.go`
`SetOfferedCurrent` calls `validateCurrent(current, 0)`; only `cmd.max_current.set` checks 6 A
(cliffhanger error-report-policy design.md, "Open points"). The text is corrected in change
`error-report-policy`; the divert itself stays.

Side effects of keeping it:
- it calls the controller outside the chargepoint service lock (the reason for several guards in
  `controller.go`: `clamp`, `offeredPause`, the second user-stop check in `dispatchPair`);
- when the pause fails, the divert logs and leaves the message to cliffhanger, which calls the
  controller a second time with the same value and logs again: one command, two attempts, two logs.

## Reproduce
Send `cmd.current_session.set_current` 0 with the Easee pause answered 500: two
`StopCharging` calls and two log lines.

## What would close it
Remove the divert and let cliffhanger's handler call the controller for every value (it already
sends the session report), then drop the guards that exist only for calls outside the service lock.
Needs a decision: the change is behaviour-neutral in theory but touches the pacing code.

## Outcome (2026-11-25)
Decided by the user: the divert is deleted (change `offered-current-normal-path`). Every value goes
through cliffhanger's handler: same pause, the report or the error from the handler. A failed pause
is now tried once and answered with one error report; before, the second call fell inside the pacing
window, was deferred and the command was answered with a success session report. No other difference
a user could see: the handler sends the same forced session report. The controller guards (`clamp`,
`offeredPause` atomic, the user-stop recheck in `dispatchPair`) stay, since session ends, observations
and the pacing timers reach the same state outside the service lock; only their comments changed.
