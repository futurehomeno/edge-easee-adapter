# A slow session resumed by a start without a mode is raised to the initial current

## What happened
Since `start-floor-every-start` every start that is not slow mode is raised to
`initial_charging_current` (16 A). Slow mode is read from the start command only; the session's
mode is recorded nowhere. Energy guard resumes a paused charger with a start without a mode, so a
session started slow at 8 A comes back at 16 A.

Found in the review of PR 237.

## Reproduce
`slowChargingCurrentInAmperes` 8, start with `charging_mode` slow (8 A), `cmd.charge.stop`, then
`cmd.charge.start` without a mode: 16 A is sent.

## What would close it
A decision: accepted (the balancer adjusts the current afterwards), or the adapter remembers that
the running session was started slow and keeps its current on a start without a mode.
