# signalr-drain-and-teardown

## Why

Two defects on the SignalR path, filed as
[#166](https://github.com/futurehomeno/edge-easee-adapter/issues/166) and
[#167](https://github.com/futurehomeno/edge-easee-adapter/issues/167).

- **Two report paths still publish inline on the drain.** `#158` moved the chargepoint and meter
  `Send*` calls onto the report queue, but `handleErrorCode` → `sendAlarmReports` and
  `handleDetectedPowerGridType` → `republishChargepointProps` still publish from
  `HandleObservation`, which is the only drainer of the observation channel. A stall on MQTT or
  inclusion fills the observation buffer, `receiver.ProductUpdate` drops observations, and the
  edge-triggered session start/stop pair is not replayed — its DB and FIMP session record is lost.
- **`Close` leaves the lifetime-energy goroutine running.** It closes the report queue and waits
  for `runSender`, but never stops `manageEnergyObservation`. `Unregister` calls `Close`, so after
  `cmd.thing.delete` that goroutine still fires at `EnergyLifetimeInterval` (10s) and calls
  `SendMeterReport` / `SendMeterExtendedReport` on a thing that was just torn down.

[#168](https://github.com/futurehomeno/edge-easee-adapter/issues/168) reported the same `notifyState`
loop as a `Timer.Stop` race — a deadline that fired while the connected state was in flight leaving a
tick in `C` for the next iteration to read. That race is real on the pre-Go-1.23 timer, but not on
this module: `src/go.mod` declares `go 1.26.0`, and from Go 1.23 the `go` directive selects unbuffered
timer channels whose pending send is discarded by `Stop`. Measured on this toolchain, 0 of 100000
stopped-after-firing timers left a readable tick under `go 1.26.0`, against 99972 of 100000 under
`go 1.22`. No change is made for it.

## What Changes

- **Alarm and inclusion reports go through the same queue as every other report.** Nothing on the
  observation drain waits on MQTT.
- **`Close` stops the energy goroutine and waits for it**, so nothing publishes on a torn-down
  thing.

## Impact

- `src/internal/signalr/client.go` — drain the deadline
- `src/internal/signalr/handler.go` — enqueue alarm/inclusion, stop the energy goroutine on Close
