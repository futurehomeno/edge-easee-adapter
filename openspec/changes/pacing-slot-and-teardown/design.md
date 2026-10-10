# Design

## Why a stored stop no longer holds the slot

An earlier revision of this change made a stored stop undisplaceable: a stop was treated as a safety
command, so a balancing refresh landing inside `offered_current_wait_time` could not cancel an
emergency pause. That rule is reversed here, deliberately, and the reversal has a cost worth writing
down rather than rediscovering.

The rule it replaces — the latest command wins, whatever its kind — is what the hub's other
subsystems assume. The charger acts on whatever it last heard, so the hub's most recent intent is the
only one worth sending. Holding an older command in the slot makes the adapter report success for a
command it then declines to carry out: `dispatch` returns "not sent", `setOfferedCurrent` turns that
into `(true, nil)`, and the caller cannot tell a dropped command from a deferred one.

### The accepted regression

A `cmd.charge.stop` stored in the slot is now cancelled by any dynamic-current write arriving before
its deadline, an emergency pause among them. This is a real loss of the property the previous
revision bought, and it is accepted rather than worked around.

It cannot be narrowed to spare emergency pauses, because "emergency-ness" never crosses the FIMP
boundary. An emergency pause arrives as an ordinary `cmd.charge.stop`; the only callers of
`StopChargepointCharging` are cliffhanger's `cmd.charge.stop` handler and the phase-mode restart, and
the alarm service is report-only. Nothing on this path distinguishes an operator's stop from a
balancer's, or an operator's start from a balancer's resume.

What bounds the exposure is the window itself: a stop is only displaceable while it sits in the slot,
at most one `offered_current_wait_time`, and only by a command the hub itself issued afterwards. A
load balancer that wants the charger paused keeps asking for a pause.

## Why a restart had to become a pair

`restartForPhaseMode` dispatches a pause and then a resume. The resume is *always* inside the window
the pause just opened, because the pause stamps `sentAt` on its way out. With one slot, the two
halves compete and exactly one survives — and both outcomes are wrong:

- stop wins: the resume is dropped and the charger stays paused for good, which is the bug this
  change fixes;
- latest wins: the pause is dropped and the mode never applies, since Easee only applies a phase mode
  at a session boundary.

So the slot carries an optional follow-up. A restart stores both halves atomically under the pacing
lock, the follow-up is promoted into the slot when the first half is sent, and the second send is
timed one full wait time later — which is what Easee's rate limit requires anyway. Any external
command discards both halves: preempting only one would either pause the charger with no resume to
follow, or resume it at a current the hub has since superseded.

## Why the restart itself stays

The bounce is not an implementation artefact that could be dropped to avoid the collision. Easee
applies a phase mode only at a session boundary, by design: to protect the charger's relays a session
completes on the phase configuration it started with, and switching requires stopping the session and
starting a new one.

A "restart only when the phase count changes" guard would be dead code. A leg-only change such as
NL1 to NL2 maps to the same Easee internal mode, so the setter's `target == current` branch returns
before the restart is reached and no stop is ever issued. Everything that reaches the restart is
already an internal-mode change — on a three-phase grid, single-phase to or from three-phase, which
is exactly where the relay constraint applies.

Source: Easee support, "The charging session changed to 1-phase and won't go back to 3-phase"
(https://support.easee.com/hc/en-gb/articles/22347392285969).
