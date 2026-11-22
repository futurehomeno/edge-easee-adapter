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
- A failed meter report no longer suppresses the state report: all three reports are attempted and
  their errors are joined.
- No state report is sent from a power reading before any charger state was observed, so a restart
  does not publish an empty state.
- cliffhanger is bumped v1.5.0 -> v1.5.3, whose PR #287 sends the forced state report before a
  failed `cmd.charge.start`/`cmd.charge.stop` returns its error (issue 19).

## Impact
- Affected specs: `reporting`
- Affected code: `src/internal/signalr/handler.go`, `src/go.mod`
