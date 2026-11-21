# Design

## Guard
One flag per charger, `unauthorizedRestarted`. It is set whenever that charger's refusal restarts the
client, for a re-sync and for a first subscribe alike, and cleared only when that charger next
subscribes successfully. A refusal restarts the client when no restart is pending on the connection
(`reconnecting`, unchanged) and the charger's flag is clear.

Setting it on a re-sync restart too keeps the existing scenario "refusal persists after the
reconnect": the charger whose re-sync restarted the client and is refused again on the new connection
does not restart a second time. A subscribed charger always has the flag clear (its success cleared
it, and a restart it triggers sets `reconnecting` until the next connect clears `isSubscribed`), so
the old `isSubscribed` condition is no longer needed.

## Why the reconnect sweep must not reset the flag
A charger that left the account is refused on every connection. Cleared by the sweep, the flag would
allow its refusal on each new connection to restart again: refused, restart, refused, restart, for
every charger on the site. Kept across reconnects, it costs one restart per adapter run; after that the
charger keeps the backoff retry it has today. A charger whose token had lapsed subscribes on the new
connection, which clears the flag, so a later lapse restarts again.

## Rejected
- No restart for a first subscribe (the status quo): the charger stays `ChargerNotSubscribed` for as
  long as the lapsed connection lives, up to hours, unless something else reconnects (issue 13).
- An unguarded restart (only `reconnecting`): `reconnecting` is per connection, so a charger refused on
  every connection restarts the client on every connection, a reconnect loop for all chargers.
