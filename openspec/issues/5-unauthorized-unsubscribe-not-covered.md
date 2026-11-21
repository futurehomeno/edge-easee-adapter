# A refused unsubscribe is not covered by the unauthorized restart

Found in the review of `resync-unauthorized` (PR #212).

`Unregister` calls `UnsubscribeCharger` on the same connection the re-sync uses. On a connection that
already refuses invocations as unauthorized, the unsubscribe of a removed charger is refused too:
`Unregister` returns the error, Easee keeps streaming that charger, and each of its observations logs
`Handle observation ... err: no handler` until the next reconnect. Only the subscribe path restarts
the client.

Open: restart on a refused unsubscribe as well, or handle the refusal once in `client.invoke` so all
three invocations share it?
