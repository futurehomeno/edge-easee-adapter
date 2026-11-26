# A command that reached Easee is answered with an error while the charger is unsubscribed

Seen on hub 9c1edbb3d437c8f1 (3.3.1, energy guard 4.6.3), 2026-11-19 16:00:40-16:02:56. EUFLF683's re-subscribe was
refused (issue 13, 135 s) while EH152843 subscribed fine on the same connection. Energy guard sent
`cmd.charge.start` three times (16:01:06, 16:01:51, 16:02:39). Each time the adapter posted the start (and the deferred
`set_current 32`) to Easee. cliffhanger v1.5.0 `handleCmdChargeStart` then calls `SendStateReport(true)`, which fails
with `charger is not subscribed for SignalR observations`, so the command is answered with `evt.error.report`.
Energy guard logged `Control span=keep_scaled_up ... rejected` three times and backed off. Once the subscription came
back at 16:02:56, the charger was charging at 32 A on 3 phases (10.75 kW). The same blindness produced
`Deferred set_current 32 was not echoed back` (3x) and `Start at 32A not echoed back within 3s`. Those are false alarms:
nothing could be echoed while the charger was unsubscribed.

Effect: energy guard's view of the charger (failed start, backoff) is the opposite of what happened (charging at full
current), for as long as the refusal lasts (about 2 min, up to 12 min with the 10 min backoff, see issue 13).

Would close it: either the follow-up reports after a successful command do not turn the reply into an error (log and
skip them, the next observation sends them), or the adapter refuses the command before it calls Easee while the
charger is unsubscribed, so the error is true. Which one is a cliffhanger question as well, since the routing is there.

Since `resync-first-subscribe-unauthorized` (2026-11-21, issue 13 archived), the first refusal of such a
charger restarts the connection once (a connectivity blip for every charger, issue 14), then it falls
back to the backoff retry. A refusal that is not a lapsed token, as here, gains nothing from that restart.
