# Charge start/stop that fails answers without a state report

## What happened

The operator decided (2026-11-20) that `cmd.charge.start` and `cmd.charge.stop` must always end in an
`evt.state.report`. Issue 18 (archived) fixed the report after a stop that succeeds. On a hard error (REST failure,
charger disconnected) cliffhanger v1.5.0 still answers with the error only (`adapter/service/chargepoint/routing.go`
returns before the state report). Energy guard record: core-energy-guard issue 059.

## What would close it

Cliffhanger PR #287 sends the state report before returning the error. Once it ships in a release after v1.5.2, bump
cliffhanger here and archive this issue.

## How to check

`go list -m github.com/futurehomeno/cliffhanger` shows the release containing #287.
