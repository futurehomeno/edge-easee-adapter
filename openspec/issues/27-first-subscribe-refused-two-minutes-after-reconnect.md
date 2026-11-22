# The first subscribe after an unauthorized restart is refused for about two minutes

Found in field logs of 3.3.1 on test sites, 2026-11-17 to 2026-11-22.

After an unauthorized restart of the SignalR client (PR #212) the first subscribe on the fresh
connection was refused as unauthorized too in 29 of 129 restarts on 8 sites. In every case Easee
stopped refusing 110-135 s after the reconnect. The change `signalr-repeated-failure-count` (packaged
`repeatedFailureCount` 8 plus a migration) only shortens the wait to the next attempt after that point,
to about 140 s; it does not remove the refusal.

The cause on Easee's side is unknown. One observation contradicts the token-lapse inference in issue
4: on a two-charger site the other charger subscribed on the same fresh connection in the same second,
and the refusal cleared by itself on that connection about two minutes later, with no new token or
reconnect.

How to observe: field logs of a hub that logs an unauthorized re-sync followed by a client restart;
look for `Unauthorized` on the first `SubscribeWithCurrentState` after the reconnect, and the time of
the first accepted subscribe for that charger. On a multi-charger site compare with the other
chargers' subscribes on the same connection.

What would close it: an explanation from Easee (or their docs) of why a hub invocation for one charger
is refused while another charger on the same connection and token is accepted, and of the ~2 min
window; or a request-side change that avoids the refusal, verified in field logs.

Related: issues 4, 14, 15.
