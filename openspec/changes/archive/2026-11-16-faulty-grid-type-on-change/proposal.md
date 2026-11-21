# Warn about a faulty grid type when it changes

## Why
- `GridType.ToFimpGridType` warns `faulty grid type detected` on every call. While a charger is idle,
  `ChargepointPhaseModeReport` falls back to the REST charger config, which runs the mapping, so a
  charger with a lasting wiring warning logs it every 30 s: 8841 times in 3 days on one 3.1.4 beta
  charger (2.8.0 logged it 870 times on the same hub, on each signalR reconnect).
- The warning itself is real (grid type 31 is an Easee wiring warning, and the grid-type alarm is
  raised for it), but repeating it buries the log.

## What Changes
- `ToFimpGridType` is a plain lookup and logs nothing.
- The grid-type observation handler warns once per charger when the detected grid type changes to a
  faulty or unknown one, with the charger ID.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/model/model.go`, `src/internal/signalr/handler.go`
