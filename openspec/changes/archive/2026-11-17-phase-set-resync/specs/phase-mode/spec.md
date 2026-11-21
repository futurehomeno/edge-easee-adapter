## MODIFIED Requirements

### Requirement: Applying A Phase Mode
`cmd.phase_mode.set` SHALL map the requested mode to an Easee internal mode and, when that target
differs from the cached internal mode, post it to `/api/chargers/{id}/commands/set_phase_mode`, then
cache the target as the internal mode, then arm a state re-sync, then record the requested FIMP mode
in the cache, then restart an in-progress session. The target SHALL be cached with the timestamp of
the internal-mode observation it replaces: the charger echoes the new mode only later, a request
arriving before that echo must be compared with the mode just sent rather than dropped as a no-op,
and every newer observation must still replace it. The re-sync SHALL be the pending one of `State
Re-sync After Start Or Stop`, requested 15 s after the send: Easee can accept a mode the charger
never applies, no observation follows then, and only the replay puts the charger's real mode back in
the cache. The request SHALL be stamped one millisecond past the newer of
the cached output-phase and internal-mode timestamps rather than with the hub clock, because those
Easee timestamps are the only values it is ever compared against. A grid with fewer than two
settable modes SHALL make the command a no-op: flipping the internal mode would bounce a session for
a change nothing can observe. When the target equals the cached internal mode the command SHALL
return successfully without calling Easee.

#### Scenario: mode changes
- **WHEN** the target internal mode differs from the cached one
- **THEN** Easee is called, the target is cached as the internal mode, the requested mode is cached
  ahead of the newest observation, and the session restart runs

#### Scenario: a second request before the echo
- **WHEN** auto (2) was just sent for a charger cached in single phase (1), no phase-mode observation
  has arrived since, and a single-phase leg is requested
- **THEN** single phase (1) is sent to Easee

#### Scenario: the charger reports after the send
- **WHEN** a phase-mode observation newer than the one the sent target replaced arrives, even one
  stamped behind the hub clock
- **THEN** it replaces the sent target, and a request for the sent mode is sent again

#### Scenario: the charger never applies the mode
- **WHEN** single phase (1) was sent for an idle charger in auto (2), Easee accepted it and no
  phase-mode observation follows
- **THEN** a re-sync is requested 15 s after the send, its replay puts auto (2) back in the cache,
  and the next single-phase request is sent to Easee again

#### Scenario: hub clock behind Easee
- **WHEN** the cached observations carry timestamps ahead of the hub clock
- **THEN** the requested mode is still stamped past them and outranks them in the next report

#### Scenario: only one settable mode
- **WHEN** the grid and phase count leave a single settable mode
- **THEN** no Easee call is made, no requested mode is recorded, and the command reports success

#### Scenario: target already in effect
- **WHEN** the target internal mode equals the cached internal mode
- **THEN** no Easee call is made and the command reports success

#### Scenario: Easee rejects the change
- **WHEN** the set-phase-mode call fails
- **THEN** the error is returned, neither the target nor the requested mode is cached, and no
  re-sync is armed
