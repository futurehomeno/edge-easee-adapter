# Design

## The session rule
"Session in progress" is the FIMP state `ready_to_charge`, `charging` or `suspended_by_ev`: the
complement of `ChargerState.IsSessionFinished`, expressed on FIMP states because the cache stores
the FIMP state. The mapper takes `lock3` and answers 3 where it answers auto; the controller passes
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
state is already the new one. An unknown or offline state triggers nothing.

## Why the lock is released at session end
The user's requirement: a lock must never be left on an idle charger, so the next car - which may
refuse three phases - starts on auto.

## Turning the setting off
The handler remembers the setting as of the previous state change. At the first change after it went
off, a charger on mode 3 gets the three-phase request, which now maps to auto. Kept in the handler,
at a state change, because the setter has no way to the chargers; the cost is that the lock lasts
until that change (the disconnect at the latest), and that an adapter restart in between forgets it
(issue 24). With the setting off all along nothing is sent, so a mode 3 set in the Easee app stays.

## Zaptec
No change: Zaptec's API has no auto mode, only `maxChargePhases` 1/3 and the installation-wide
`threeToOnePhaseSwitchCurrent`.
