# Design

## The session rule
"Session in progress" is the FIMP state `ready_to_charge`, `charging`, `suspended_by_ev`,
`finished`, `error` or `requesting`, expressed on FIMP states because the cache stores the FIMP state. `finished` (Easee
"Completed") counts as in progress (the user, 2026-11-22, issue 23): Easee reports it for a
plugged-in car that is full or paused, and releasing there made every top-up start on auto and pay
a stop/resume to lock again. Only `disconnected` releases the lock (the user, 2026-11-22): `error`
and `requesting` (awaiting authentication) keep it, since the car is still plugged in, and
`requesting` after `disconnected` is the session start. The list stays positive so the empty state
of an empty cache is no session; an unknown or offline state moves nothing. Easee may report `error`
with no car plugged in (issue 30). The mapper takes `lock3` and answers 3 where it answers auto; the controller passes
`use_locked_3_phases && session in progress` from the cached state, for the send and for
`requestStillHolds`, so the echo of the lock keeps answering the report.

## Why the lock is applied at session start
The hub does not repeat a request for a mode the report already shows. A charger left on auto
between sessions reports the three-phase mode, so without a trigger of its own the lock would last
one session. On a FIMP state change, the handler queues `SetPhaseMode(<three-phase mode>)` on the
chargepoint service when the session starts on auto or ends on mode 3; the controller then picks 3
or 2 from the new cached state. Going through the service takes the lock that guards the
controller's `sentPhase` fields and validates the mode, and needs no new controller method. The call
is queued after the state report, on the sender, so the report is not held behind it and the cached
state is already the new one. The queued call checks the internal mode again, so a mode set while
it waited, such as single phase, stays; a hub request that takes the service lock between that check
and the call is still overwritten. An unknown or offline state triggers nothing.

## Why the lock is released at session end
The user's requirement: a lock must never be left on an idle charger, so the next car - which may
refuse three phases - starts on auto. For the same reason the first state after an adapter restart
releases a mode 3 too, when it is `disconnected`.

## A setting changed during a session
Forces nothing (the user, 2026-11-22, issue 24): the handler acts only when a session starts or the
new state is outside a session, never on a change from one session state to another, and it keeps its own "in
session" flag so that an offline charger's unknown state does not make the rest of the session look
like a new one. The new value is used by the next request that takes the charger from single phase
to three phases, and from the session end on. A session that started with the setting on is still
released at its end when the setting is off by then: the handler remembers the setting as of the
session start. With the setting off all along nothing is sent, so a mode 3 set in the Easee app
stays. A mode 3 observed while the setting is on is remembered the same way, for a lock a hub
request placed in a session that started with the setting off; an observation the cache turns away
as older counts for nothing, and so does one that arrives after the setting went off again. The
memory ends when the release goes out, so a mode 3 set elsewhere afterwards with the setting off
stays; a release that fails, finds the report queue full or is skipped because fewer than two modes
are advertised keeps it, and is asked again at the next session start or end. A mode 3 observed
after the release, before the charger shows auto, sets it again, and the observation of auto does
not end it (issue 29, accepted by the user 2026-11-22). That memory is in the handler: an adapter
restart between turning the setting off and the session end leaves mode 3 on the charger until the
next single-to-three-phase request (issue 31, accepted by the user 2026-11-22).

The lock at the session start is sent once (issue 28, accepted by the user 2026-11-22): a failed
send, or a start while a topology update is pending, leaves that session on auto.

## No stop and resume at the disconnect
The release is written right after the `disconnected` observation. A power reading stamped at or
after it makes the reported state `charging`, and the controller would stop and resume a charger no
car is on, with the resume landing on the next car. The controller skips the bounce for a
`disconnected` charger. Only for that state (the user, 2026-11-22): a guard on every state outside
a session would also drop the bounce right after an adapter restart, when the cached state is
still empty.

## Zaptec
No change: Zaptec's API has no auto mode, only `maxChargePhases` 1/3 and the installation-wide
`threeToOnePhaseSwitchCurrent`.
