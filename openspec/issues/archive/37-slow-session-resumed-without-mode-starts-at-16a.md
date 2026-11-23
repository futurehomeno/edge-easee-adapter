Resolved: 2026-11-23, the operator: a slow session that energy guard pauses and resumes comes back at its slow current. The cache remembers the session's last explicit start mode; a start without a mode takes it; the session-finished observation forgets it. Change `slow-session-resume`; the restart gap is issue 38.

# A slow session resumed by a start without a mode comes back at 16 A

## What happened
Found in the review of PR 237 (`start-floor-every-start`). Energy guard pauses a charger with
`cmd.charge.stop` and resumes it with a bare `cmd.charge.start` with no mode
(`core-energy-guard` `devicemanager/device.go`, `requestScaleUp`/`requestRevert`). The adapter does
not record which mode a session was started in. So after PR 237 a slow session that energy guard
paused and resumed is raised to `initial_charging_current` (16 A, capped by the max current). Before
PR 237 it resumed at the cached current.

## How to reproduce
Start in mode `slow`, with `slowChargingCurrentInAmperes` set or 0 and a cached current of 6 A. Send
`cmd.charge.stop`, then `cmd.charge.start` with no mode. The charger is offered 16 A.

## Doubt
The operator's rule (2026-11-23) is: every start that is not slow mode starts at 16 A, capped by the
maximum. A bare start is not slow mode, so this follows the rule. Zaptec raises on every start, slow
mode included. Is a slow session resumed by energy guard meant to come back at 16 A?

## What would close it
- The operator confirms 16 A is fine here: archive this issue. Or
- remember the last explicit start mode per charger and treat a bare start as that mode (a small
  piece of controller state, cleared when the session ends).
