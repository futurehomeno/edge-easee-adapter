## Context

The adapter learns charger state only from Easee's SignalR `ProductUpdate` stream. Easee sends a full snapshot when a charger is subscribed with `SubscribeWithCurrentState(id, true)`, and after that only changes. When the stream stops carrying a value, the cache keeps the old one forever.

## Decisions

### Re-sync through `SubscribeWithCurrentState`, not REST

The re-sync re-invokes `SubscribeWithCurrentState(id, true)` on the live connection for a charger that is already subscribed. This is the path that repaired the charger on the reported hub (the restart at 00:58:47), and its snapshot goes through the same handlers, timestamp guard and reports as every other observation. Joining a SignalR group the connection is already in is idempotent on the server.

Alternative: `GET /api/chargers/{id}/state` over REST. It needs a new client call and a mapping from REST fields onto observations, and its values would bypass the observation timestamp guard. It stays an option if a re-subscribe turns out not to replay the state.

### Triggers

- **After an accepted start or stop**, 15 s after the command was sent: long enough for the charger to act, and well inside energy-guard's 60 s wait. Only one re-sync per charger is pending at a time; a later command pushes it back.
- **Charging watchdog**: checked every minute. It fires when the cached state is `charging` and no `TotalPower` or `InCurrentT3..T5` observation has arrived for 5 minutes, and again at most every 5 minutes while that lasts. The `DynamicChargerCurrent` echo of our own commands does not count, because it kept arriving during the reported outage.

### Power override follows observation order

A cached `TotalPower > 0` forces `charging` only when its observation timestamp is not before the charger-state timestamp. A charger that reports power while its state lags is still `charging`, and a state observation newer than the last power reading wins.

## Risks

- A charger whose Easee stream never carries `TotalPower` or phase currents while charging is re-synced every 5 minutes. A snapshot is one small burst per charger, so this is accepted.
