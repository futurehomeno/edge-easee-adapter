## Context

`resyncStale` runs every minute. `refreshesState` counts `ChargerOPState` and `TotalPower` above zero
as activity, because in the fault of issue #178 the stream still carried `DynamicChargerCurrent` echoes
and phase currents while the state was stale.

## What Easee streams at a steady current

Debug logs, with the observations replayed by a re-subscribe left out:

| Hub | Chargers | Charging hours | Logged re-syncs |
|---|---|---|---|
| 9c1edbb3d437c8f1 (2026-11-14 to 11-21) | EH152843, EUFLF683 | 63 | 258 |
| beta hub B (2026-11-16 to 11-19) | B1, B2 | 16 | 97 |
| beta hub C (2026-11-17 to 11-25) | C1 | 11 | 72 |

- `TotalPower` comes on a change of the power and, when it does not change, again 900-902 s after the
  previous one: 10 such gaps on hub 9c1edbb3d437c8f1 (EUFLF683, 2026-11-16: 22:56:29, 23:11:29,
  23:27:03, 23:42:04). Easee's observation list calls it "Telemetry" and gives no interval, so the 15
  minutes are observed, not documented.
- Longer gaps were all stream faults: 36 min with failed subscribes and a disconnect (EH152843,
  2026-11-16 00:53-01:29), 17 min while the charger was refused after an unauthorized restart
  (B2, 2026-11-18 00:13-00:31).
- `EnergySession` comes once per kWh (9.6 min at 6.3 kW), phase currents on small changes, minutes
  apart and irregular. `ChargerOPState` comes on a state change only.

## Decision: a 20-minute window

The window must outlast the 15-minute resend; 20 minutes leaves room for a late one. Fires of the
watchdog replayed over the logs above, all five chargers:

| Counts as activity | 5 min | 10 min | 15 min | 20 min |
|---|---|---|---|---|
| state, power (as now) | 599 | 180 | 108 | 4 |
| state, power, phase current of 1 A or more | 519 | 145 | 44 | 3 |
| state, power, session energy grown | 487 | 113 | 41 | 3 |

The replay over-counts the 5-minute case (427 were logged); the ratio is what matters.

Alternatives:
- Count more observations as activity. It does not stop the fires at any window under 15 minutes,
  and phase currents are what #178 excluded.
- Read the state over REST, as Zaptec does. No hub invocation, so no refusal, but it needs a new
  client call and a mapping onto observations (rejected for the same cost in
  `resync-state-after-charge-command`).
- Reconnect when the token is renewed. That is the same disconnect every hour, on every site.

## Risks
- Easee may resend less often on other charger models or firmware. Then the watchdog fires every 20
  minutes there, a quarter of today's rate.
- A charger that stops by itself while its stream is silent stays `charging` for up to 21 minutes. A
  stop the adapter sent is still re-synced after 15 s.

## Also seen: when the refusal comes

Hub 9c1edbb3d437c8f1, 2026-11-20 19:21 to 2026-11-21 08:26: the authenticator logged in again 14
times, about an hour apart. Each of the first 13 was followed by exactly one refused re-sync, 91 s
to 38 min later (00:26:23 login, refused 00:27:54); a re-sync 84 s after a login was still accepted
(02:26:30 login, accepted 02:27:54, refused 02:34:54). The
refusal follows the end of the token the connection was opened with, not the age of the connection.
Recorded in issue 4.
