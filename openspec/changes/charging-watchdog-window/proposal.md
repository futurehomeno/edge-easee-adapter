# The charging watchdog waits 20 minutes, not 5

## Why
- The watchdog re-syncs a charger cached as `charging` when no `ChargerOPState` and no `TotalPower`
  above zero arrived for 5 minutes. Easee sends neither that often while a car charges at a steady
  current, so the watchdog fires every 6-17 minutes through a healthy session (issue 15).
- A re-sync is a hub invocation on the connection's token. Once that token has ended (the adapter logs
  in again every hour) the re-sync is refused as unauthorized and the manager restarts the SignalR
  client: every charger of the site is reported disconnected for a moment (issue 14), and after about
  one restart in five the first subscribe is refused for two more minutes (issue 27).
- 3.3.1 beta logs of 2026-11-25: 63 such restarts on 7 of 98 sites, one site restarting at 23:27,
  00:36, 01:36 and 02:33 during one overnight session.
- Debug logs of three hubs (five chargers, about 90 charging hours) show that Easee resends
  `TotalPower` no later than 900 s after the last one while charging. Replayed against them, the
  5-minute check fires 599 times and a 20-minute check 4 times; the two of those four that were
  looked at were real stream faults (design.md).

## What Changes
- `staleStateWindow` 5 min -> 20 min: the silence before the watchdog re-syncs, and the shortest time
  between two watchdog re-syncs.
- Nothing else: the observations that count as activity, the one-minute check, and the re-sync 15 s
  after an accepted start or stop (the case of issue #178) are unchanged.

## Impact
- Affected specs: `charging-control` (Charging Watchdog).
- Affected code: `src/internal/signalr/manager.go` (one constant).
- A stream that went silent while the charger really stopped by itself is noticed after up to 21
  minutes instead of 6. Until then the state report says `charging` with the last power, as it did
  for up to 6 minutes before.
- A re-sync after a start or stop on a connection whose token has ended still restarts the client
  (issue 14 stays open).
