# Keep retrying a refused subscribe every 30 s for 4 min 20 s

## Why
- When a re-sync is refused as unauthorized the manager restarts the SignalR client (PR #212; since
  3.4.0 also for a first subscribe, once per charger). Field logs of 3.3.1 on test sites, 2026-11-17
  to 2026-11-22, show 129 such restarts on 8 sites.
- After 29 of them the first subscribe on the fresh connection was refused as unauthorized too. The
  charger then waits on the retry backoff, unsubscribed: no observations, every report fails with
  "charger is not subscribed for SignalR observations", and energy guard's commands get no state
  report back (on one site `cmd.charge.stop` was repeated six times in ten minutes).
- In all 29 cases Easee stopped refusing 110-135 s after the reconnect.
- With the packaged settings (10 s ×2, 30 s ×3, then 10 min) attempts run at 0, 10, 20, 50, 80, 110 s
  and then at 710 s: 6 cases on 2 sites missed the end of the refusal by seconds and stayed
  unsubscribed for 710 s. Hubs carrying older stored settings happened to try at 135 s and recovered
  there (23 cases, 6 sites).
- 3.4.0's `unauthorizedRestarted` guard does not help: the one restart per charger is already spent,
  so the charger still falls on the backoff.

## What Changes
- The packaged `signalR.repeatedFailureCount` goes from 3 to 8. Attempts then run at 0, 10, 20, 50,
  80, 110, 140, 170, 200, 230, 260 s and every 10 min after: recovery expected at 140 s instead of
  710 s.
- A config migration to version 10 lifts a stored `signalR.repeatedFailureCount` below 8 to 8 and
  leaves 8 or more alone. The packaged defaults only seed a fresh install; an installed hub's stored
  value overrides them, so without the migration existing hubs keep 3.
- The backoff stays cliffhanger's three-step stateful backoff; durations are unchanged.

## Impact
- Affected specs: `configuration`
- Affected code: `src/internal/config/config.go`, `src/cmd/factory.go`, the packaged `defaults/config.json`
- Side effect: the same settings drive the client's redial after a lost connection
  (`src/internal/signalr/client.go`, `cfg.SignalRBackoffStateful()`), so the redial also retries every
  30 s for 4 min 20 s before dropping to 10 min, instead of 1 min 50 s.
- Side effect: a charger that stays refused (it left the account) costs five more subscribe attempts
  per connection, then 10 min as before.
- A count below 8 set on purpose over `cmd.config.set_signalr_repeated_failure_count` is lifted too.
- Not fixed: the roughly two-minute refusal itself; its cause on Easee's side is unknown (issue 27).
