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
