# A calmer SignalR backoff: 3 × 60 s, then 3 × 5 min, then every 30 min

## Why
- When a re-sync is refused as unauthorized the manager restarts the SignalR client (PR #212; since
  3.4.0 also for a first subscribe, once per charger). Field logs of 3.3.1 on test sites, 2026-11-17
  to 2026-11-22, show 129 such restarts on 8 sites.
- After 29 of them the first subscribe on the fresh connection was refused as unauthorized too. The
  charger then waits on the retry backoff, unsubscribed: no observations, every report fails with
  "charger is not subscribed for SignalR observations", and energy guard's commands get no state
  report back (on one site `cmd.charge.stop` was repeated six times in ten minutes).
- In all 29 cases Easee stopped refusing between 110 s and 135 s after the reconnect.
- With the 3.4.0 packaged settings (10 s ×2, 30 s ×3, then 10 min) attempts run at 0, 10, 20, 50, 80,
  110 s and then at 710 s. Hubs carry older stored settings too, some retrying as often as every 5 s.
  Retrying every 10-30 s is more than Easee's server should take, so the schedule gets calmer rather
  than denser.

## What Changes
- Packaged SignalR backoff: `initialBackoff` `60s` × `initialFailureCount` 3, `repeatedBackoff` `5m`
  × `repeatedFailureCount` 3, `finalBackoff` `30m`. Subscribe attempts after a reconnect: 0, 60, 120,
  180 s, then 480, 780, 1080 s, then every 30 min. Steps of 60 s rather than 30 s put the 180 s
  attempt (and the 120 s one, when the refusal has ended by then) after the 110-135 s refusal, for
  the same number of calls.
- A config migration to version 10 sets all five stored SignalR backoff values to these, whatever
  they were: the packaged defaults only seed a fresh install and an installed hub's stored values
  override them. A value an operator set on purpose is overwritten too.
- The getters' and the stateful backoff's fallbacks become 60s / 5m / 30m, equal to the packaged values.
- The backoff stays cliffhanger's three-step stateful backoff. The authenticator backoff is unchanged.

## Impact
- Affected specs: `configuration`, `observation-streaming`
- Affected code: `src/internal/config/config.go`, `src/cmd/factory.go`, the packaged `defaults/config.json`
- Recovery after a refused first subscribe: 120 to 180 s. Hubs on the 3.4.0 packaged settings improve
  from 710 s (6 of the 29 cases, 2 sites). Hubs on older stored settings, which today recover at 135 s,
  land at 120 to 180 s (23 cases, 6 sites).
- The client's redial after a lost connection follows the same settings
  (`src/internal/signalr/client.go`): 60, 120, 180 s, then every 5 min three times, then every 30 min,
  where today it is 10, 20, 50, 80, 110 s, then every 10 min. After a long outage the reconnect can
  come up to 30 min after the network is back.
- A charger that stays refused (it left the account) costs 7 attempts in the first 18 min, then one
  every 30 min.
- The 110-135 s window comes from 29 cases over five days; a refusal lasting longer than 180 s would
  wait until 480 s.
- Not fixed: the roughly two-minute refusal itself; its cause on Easee's side is unknown (issue 27).
