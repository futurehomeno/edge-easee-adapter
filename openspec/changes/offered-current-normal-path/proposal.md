# Offered current on the normal path

## Why
A routing step (`routeOfferedPause`, `divertOfferedPause`) handed `cmd.current_session.set_current`
values from 0 to 5 A to the controller before cliffhanger's chargepoint handler, renamed the message
so the handler skipped it and sent the session report itself. It was written because cliffhanger was
believed to reject a value under 6 A; it does not (issue 56). The handler calls the same
`SetChargepointOfferedCurrent`, which already pauses under `min_charging_current`. When the pause
failed, the step logged and left the message to the handler, which called the controller again: the
second call fell inside the pacing window, was deferred and the command was answered as a success.

## What Changes
- The routing step, its route and its tests go; every value takes cliffhanger's handler.
- An offered current under the minimum still pauses charging, and the session report comes from the handler.
- A failed pause calls the controller once and is answered with one error report.
- The routing `Controller` interface keeps only `Bind` and `Unbind`.

## Impact
- `src/internal/routing/router.go`, `src/internal/easee/controller.go` (comments), `src/cmd/easee_internal_test.go`.
