# Charge stop: the state leaving `charging` is not reported while power is still above zero

## What happened

Hub 9c1edbb3d437c8f1 (Easee 3.3.1, energy guard 4.6.3), 2026-11-18 17:42:40 (also 18:05:40, 21:22:40): energy guard
sent `cmd.charge.stop` (Max Guard pause). The Easee showed `await_start` about 2 s later, but energy guard never got an
`evt.state.report` leaving `charging`, only a 0 W power reading. It sent a second stop 12 s later; the adapter delayed it
(`stop, delayed 18s`), and on 2026-11-17 07:18 such a deferred stop came back as 400 `ChargerDisconnected`. Energy
guard's record: core-energy-guard `openspec/issues/059-second-charge-stop-after-max-guard-pause.md`.

The same thing is in the local log of that hub, 2026-11-17 17:40 (`~/fimpee/logs_9c1edbb3d437c8f1_20261118`):
EUFLF683 got `TotalPower=1.08kW` at 17:40:42 and `State=await_start` at 17:40:43; energy guard [033] still read
`charging=true` at 17:40:49 after `TotalPower=0.00kW`, and sent the second stop at 17:40:53 (`stop, delayed 17s`).
EH152843 got power 0 and `await_start` in the same second, so energy guard saw `charging=false` and did not repeat.

## What the code does today (origin/release_3.4, same as v3.3.1)

- On a successful `cmd.charge.start`/`cmd.charge.stop`, cliffhanger v1.5.0 always sends a forced
  `evt.state.report` (`adapter/service/chargepoint/routing.go:75` and `:111`). It is read right after the command, so it
  carries the state before the charger acted, normally `charging` for a stop.
- The later change is reported only from the SignalR state observation, on change only:
  `src/internal/signalr/handler.go:443` `SendStateReport(false)`.
- The reported state is `cache.ReportedState` (`src/internal/cache/cache.go:132-140`): any power reading above 0 that is
  not older than the state turns it into `charging`. So `await_start` arriving with residual power reads `charging`, equal
  to the last report, and is dropped as unchanged.
- The power observation (`handler.go:463-492`) sends only meter reports, never the state report. When power reaches 0
  the state is not re-evaluated, so `charging` stays the last report until the next state observation or the re-sync
  15 s after the command (`src/internal/easee/controller.go:31`, `:298-300`). Easee has no periodic chargepoint report task.
- A stop inside the pacing window is stored (`controller.go:186-197`, "stop, delayed Ns"); the command answers at once
  and the forced report shows the unchanged state. The deferred send's result is only logged (`controller.go:354-356`):
  a failure (400 `ChargerDisconnected`) produces no report and no re-sync.
- A start or stop that returns an error (REST failure, `charger removed`) answers with the error only: cliffhanger returns
  before the state report (`routing.go:70-73`, `:106-109`).

## What would close it

The operator's rule (2026-11-20): `cmd.charge.start` and `cmd.charge.stop` always end in an `evt.state.report`, as FH
Charge in zigbee-ad does (its device answers the command with its new state, `fh_charge.json:453`, decoded in
`services/instances/chargepoint/func.go:55-81`).

Smallest change: in `handleTotalPower` also enqueue `chargepointSrv.SendStateReport(false)` after the meter reports. The
power reading that ends the `charging` override then publishes `await_start` (or `charging` on a start); the on-change
dedup keeps it to real changes. Risk: low; during a ramp-down a late reading above 0 can briefly report `charging` again
before 0 W, which is what the cached value already says today.

For an error answer, the forced state report would have to move in front of the error return in cliffhanger
(`handleCmdChargeStart`/`handleCmdChargeStop`, keeping the original error). That is a shared-library change for every
chargepoint adapter; decide with Zaptec (its issue 014).

## How to check

- Unit test in `signalr/handler_test.go`: reported state `charging`, then a state observation `await_start` with a power
  reading > 0 stamped at or after it, then `TotalPower=0`; expect an `evt.state.report` `await_start` after the 0 W reading
  (fails today).
- On a hub at DEBUG: after an energy guard stop, the adapter's FIMP output shows `evt.state.report` leaving `charging`
  within a few seconds of `TotalPower=0.00kW`, and no second `cmd.charge.stop` / `stop, delayed` follows.
