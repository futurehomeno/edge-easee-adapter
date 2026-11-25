# A first subscribe on a token-lapsed connection is refused until something else reconnects

Found in the review of PR #213.

The unauthorized restart fires only for a charger already subscribed on the connection (the loop
guard in `resync-unauthorized`). A charger registered on a connection whose token has lapsed, or
repaired there by `repairReplacedSubscription`, is refused on every backoff retry (up to 10 min
apart) and stays `ChargerNotSubscribed` until something else reconnects: another charger's refused
re-sync, a network drop or an adapter restart. This hits multi-charger sites that add or re-select a
charger more than ~1 h after the last connect, and rests on the inferred token-lapse cause (issue 4).

Open: restart for a first subscribe too? That needs a per-charger guard, because a charger that left
the account is refused on every new connection and would loop the reconnect for all chargers. The
unexplained refusal right after connect would then cost an extra reconnect each time. Same gap as
issue 5 for a refused Unsubscribe.

Field data, 3.3.1 beta (rollo logs 2026-11-19): after a refusal, the refused charger stayed unsubscribed
for about 710 s while charging. That happened 3 times each on hubs c2ff195b and ac369a68 and once on
e434ffd8; on 8ee88ca8 the gaps were about 135 s. On c2ff195b at 21:51:45, EHTSKCHA was refused again on
the fresh connection while EHAZ4B5Y subscribed on it fine. So that refusal was specific to the charger,
not a lapsed token. Its retries ran at 10-30 s, then backed off to 10 min (resubscribed at 22:03:35). EG's
three `cmd.charge.start` in that window all came back as errors.

How long a refusal lasts: on 8ee88ca8 (debug log), the refusals at 16:47:54, 23:59:21 and 06:09:06 each
kept failing at +5, +10, +15, +45, +75 and +105 s, and were accepted at +135 s. On c2ff195b the refusal
still failed at +110 s; by then the retries had backed off to 10 min, so the next try was at +710 s. So
a refused charger is refused for about 2 min, and a single retry 10 s later would not have helped in any
of the observed cases. The 12 min gaps come from the jump to the 10 min backoff, not from the refusal
itself.

Closed by `resync-first-subscribe-unauthorized` (2026-11-21): a first subscribe refused as unauthorized restarts the client too, at most once per charger until it next subscribes (the reconnect sweep keeps that guard).
