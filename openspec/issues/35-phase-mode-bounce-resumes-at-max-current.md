# The phase-mode bounce can resume a session at the max current

## What happened
`restartForPhaseMode` (`src/internal/easee/controller.go`) resumes at the requested offered
current, then the observed offered current, then `MaxCurrent()`. With the first two unknown it
resumes at the charger's max, the over-offer issue 33 closed for `cmd.charge.start`.

Found in the review of PR 236; the code is older than that PR.

## Reproduce
Adapter restarted during a session, before the offered-current observation is replayed;
`cmd.phase_mode.set` bounces the session and resumes it at the cached max current.

## What would close it
The last fallback is `initial_charging_current` capped at the max, as for a start, or a recorded
reason why the max is right there.
