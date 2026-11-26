# The stale-state watchdog fires every 6-9 min during steady charging

Seen on the 3.3.1 rollout (logs 2026-11-17 16:24-18:00, rollo `by-hub-3`), and already on the 3.2.5 beta
(2026-11-14: up to 309 `No state or power for 5m0s while charging, re-syncing` per hub).

`resyncStale` re-syncs a charging charger when no `ChargerOPState` and no `TotalPower > 0` observation has
arrived for `staleStateWindow` (5 min). While a session charges at a steady current, Easee does not stream
either of them that often:

- hub 05ff3cfb0b663d0d3b, EHHNELUW: charging from 17:00:34, re-synced at 17:07, 17:13, 17:22, 17:29, 17:37,
  17:43, 17:52 and 17:58. That is 8 re-subscribes in 58 min.
- hub 05fea5af0a88336c17, EH9PSYRA: 4 re-syncs during a 52 min session that delivered 6.3 kWh, so it was
  really charging.

Every re-sync is a `SubscribeWithCurrentState` and a replay of the last session. About an hour after
connect, the re-subscribe is refused as unauthorized and the client restarts (issues 4 and 14). On both hubs
this happened at 17:21 and 17:22, 57-58 min after the 16:24 connect, and caused a connectivity blip in the
middle of the session.

Open:
- How often does Easee stream `TotalPower` and `ChargerOPState` during steady charging? Answered below.
- Should the window be longer, should other observations reset it (`DynamicChargerCurrent` echoes, phase
  currents, lifetime energy), or both? #178 is why those observations do not reset it now: they kept
  streaming while the state did not.
- Every upgraded hub restarted at the same minute, so their ~1 h token lapses line up as well. At 17:24 hub
  05ff3bf0ea5d72dee3 reconnected and both REST calls of its phase-mode report timed out
  (`Client.Timeout exceeded while awaiting headers`). That was the only time this happened.

Answer to the cadence question, hub 9c1edbb3d437c8f1 (debug log, 3.3.1, logs to 2026-11-19 16:14): Easee streams
`TotalPower` only when the power changes, 2-30 s after an offered-current change, and `ChargerOPState` only on a state
change. Neither is a heartbeat. EUFLF683, 2026-11-18 22:00-23:50: 36 `TotalPower` in 110 min, every one right after an
`Offered current` change or a re-sync replay; 22:11:58 -> 22:17:54 had none, and the watchdog fired. In the window
2026-11-18 03:15 -> 2026-11-19 16:14 the two chargers were re-synced 55 times, and 20 of the re-subscribes were refused
as unauthorized (restart, issue 14). So a window of any fixed length fires whenever the current is held steady for that
long. A longer window only spaces the re-syncs out.

2026-11-25, 3.3.1 beta logs (rollo `logs/beta/2026-11-25/easee`): 63 unauthorized restarts on 7 of 98 sites, 54 of
them on 3 sites. One site, one charger charging overnight: watchdog re-sync every 6-17 min, refused and
restarted at 23:27, 00:36, 01:36, 02:33. `staleStateWindow` and `resyncStale` are unchanged in 3.4.2 and on
release_3.4, so the hourly restart is still there.

Closed by `charging-watchdog-window` (2026-11-25): the window is 20 minutes. Easee resends `TotalPower` no later
than 900 s after the last one while charging (observed on three hubs, not documented), so a healthy session no
longer trips the watchdog. Other observations do not reset it, as before.
