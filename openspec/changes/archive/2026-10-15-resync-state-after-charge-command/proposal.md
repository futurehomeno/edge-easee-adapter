## Why

Issue: #178.

On hub `7a18325dc9eaa3b7` (Easee `EH496315`, v3.1.4, 2026-09-24 00:46–00:58) energy-guard's Max guard sent `cmd.charge.stop` every 1–2 minutes for 12 minutes, and every stop timed out. Each stop was accepted by the Easee cloud (`wasAccepted: true`), and the charger did stop. When the adapter restarted at 00:58:45 and re-subscribed, Easee's snapshot showed `State=await_start`, `TotalPower=0.00kW` and `i3=0.1`, and the house meter had shown about 2 A on L3 since 00:46.

After the stop, the live SignalR stream stopped delivering `ChargerOPState`, `TotalPower` and `InCurrentT5` for that charger. It carried only the `DynamicChargerCurrent` echo of each command, plus `i1`/`i2` once. The adapter's cache therefore kept `charging`, 19.17 A on L3 and the last `TotalPower`, and nothing in the adapter ever refreshes or ages them:

- The forced `evt.state.report` that cliffhanger sends after `cmd.charge.stop` is built from that cache, so it says `charging`. A cached `TotalPower > 0` forces `charging` whatever the cached state or the age of the power reading.
- energy-guard waits 60 s for a non-charging state or a power reading under 500 W, times out and backs off. Its periodic `cmd.meter.get_report` is answered from the cache under a fresh message timestamp, so energy-guard's own 45-minute power-freshness bound never lapses.
- Only a re-subscribe (`SubscribeWithCurrentState`), which today happens only after a reconnect or a restart, brings back the real state.

The Zaptec adapter handles the same class of fault with a 5-minute charging watchdog that re-reads the charger over REST (`chargingStaleWindow`).

## What Changes

- After an accepted start or stop, the adapter re-syncs that charger's state from Easee after a short delay, well inside energy-guard's 60 s wait.
- While a charger is cached as charging and no state or power observation has arrived for 5 minutes, the adapter re-syncs it. The window matches Zaptec's `chargingStaleWindow`.
- A re-sync asks the live SignalR connection for the charger's current state (`SubscribeWithCurrentState`), even when the charger is already subscribed. The snapshot flows through the existing observation handlers and reports.
- A cached `TotalPower > 0` forces `charging` only while the power observation is not older than the charger-state observation.

## Impact

- `internal/signalr`: an on-demand re-sync request that bypasses the already-subscribed skip, and the charging watchdog.
- `internal/easee`: re-sync scheduling after start and stop, and the state-report power override.
- No config, FIMP or API change.
