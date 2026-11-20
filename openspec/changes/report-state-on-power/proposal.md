# Report the charger state on power readings

## Why
Issue 18: after an energy guard `cmd.charge.stop` the Easee reports `await_start` while the total
power is still above zero. The reported state stays `charging` (a power reading not older than the
state wins), so the unforced state report sees no change and is dropped. When the power reading
then drops to 0 W only meter reports are published, so `charging` stays the last report and energy
guard sends a second stop. The operator's rule (2026-11-20): `cmd.charge.start` and
`cmd.charge.stop` always end in an `evt.state.report` of the state after the charger acted.

## What Changes
- A total-power observation that updates the cache also publishes an unforced chargepoint state
  report after the meter reports. The on-change filter keeps it to real changes, so the 0 W reading
  that ends the `charging` override publishes the cached state (`ready_to_charge` after a stop).
- Error answers to start/stop (no state report from cliffhanger) are out of scope: that is a
  shared-library change.

## Impact
- Affected specs: `reporting`
- Affected code: `src/internal/signalr/handler.go`
