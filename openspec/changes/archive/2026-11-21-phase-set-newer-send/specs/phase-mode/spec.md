## MODIFIED Requirements

### Requirement: Applying A Phase Mode
`cmd.phase_mode.set` SHALL map the requested mode to an Easee internal mode and, when that target
differs from the cached internal mode or from a target sent less than 15 s before, post it to
`/api/chargers/{id}/commands/set_phase_mode`, then cache the target as the internal mode, then record
the requested FIMP mode in the cache, then restart an in-progress session, or arm a state re-sync
when there is none to restart. The target SHALL be cached with the timestamp of the internal-mode
observation it replaces: the charger echoes the new mode only later, a request arriving before that
echo must be compared with the mode just sent rather than dropped as a no-op, and every newer
observation must still replace it. The re-sync SHALL be the pending one of `State Re-sync After
Start Or Stop`: Easee can accept a mode the charger never applies, no observation follows then, and
only the replay puts the charger's real mode back in the cache. It SHALL be requested 15 s after the
send, unless one is already pending, which SHALL be kept as it is: phase sets are not paced, so
replacing it could hold back the re-sync of a start or a stop for as long as they keep coming. A
restarted session SHALL arm none at the phase send: its stop and its resume each arm one once sent,
and one armed earlier would run before a stop the pacing window stored. The request SHALL be stamped
one millisecond past the newer of the cached output-phase and internal-mode timestamps rather than
with the hub clock, because those Easee timestamps are the only values it is ever compared against.
A grid with fewer than two settable modes SHALL make the command a no-op: flipping the internal mode
would bounce a session for a change nothing can observe. When the target equals the cached internal
mode the command SHALL return successfully without calling Easee, unless a different target was
accepted by Easee less than 15 s before, measured on the hub clock: the cache has then moved off that
send (an older echo, a replay, or a change made elsewhere) while the charger may still apply it, so
the target SHALL be sent as if it differed. The last sent target SHALL only veto that skip, never
stand in for the cached internal mode, and it SHALL lapse after 15 s, when the re-sync has replayed
the charger's real mode. A mode changed elsewhere within 15 s of a send and then requested is
therefore sent again, bouncing a charging session once more.

#### Scenario: mode changes
- **WHEN** the target internal mode differs from the cached one
- **THEN** Easee is called, the target is cached as the internal mode, the requested mode is cached
  ahead of the newest observation, and the session restart runs

#### Scenario: a second request before the echo
- **WHEN** auto (2) was just sent for a charger cached in single phase (1), no phase-mode observation
  has arrived since, and a single-phase leg is requested
- **THEN** single phase (1) is sent to Easee

#### Scenario: a third request between two echoes
- **WHEN** auto (2) and then single phase (1) were sent for a charger cached in single phase, the
  auto echo lands newer than the observation the single-phase target replaced, and a three-phase
  mode is requested within 15 s of the single-phase send
- **THEN** auto (2) is sent to Easee

#### Scenario: a replay before the echo
- **WHEN** single phase (1) was sent for a charger cached in auto (2), a replay puts auto back in the
  cache within 15 s of the send, and a three-phase mode is requested
- **THEN** auto (2) is sent to Easee

#### Scenario: the last sent mode is requested again
- **WHEN** single phase (1) and then auto (2) were sent less than 15 s ago and the cache holds auto
- **THEN** a three-phase request makes no Easee call: only the last send vetoes the skip

#### Scenario: the charger reports after the send
- **WHEN** a phase-mode observation newer than the one the sent target replaced arrives, even one
  stamped behind the hub clock
- **THEN** it replaces the sent target, and a request for the sent mode is sent again

#### Scenario: the charger never applies the mode
- **WHEN** single phase (1) was sent for an idle charger in auto (2), Easee accepted it and no
  phase-mode observation follows
- **THEN** a re-sync is requested 15 s after the send, its replay puts auto (2) back in the cache,
  and the next single-phase request is sent to Easee again

#### Scenario: the replay restored the mode after the window
- **WHEN** single phase (1) was sent for an idle charger in auto (2), the charger never applied it,
  and the re-sync 15 s after the send put auto (2) back in the cache
- **THEN** a three-phase request makes no Easee call

#### Scenario: the session restart is stored
- **WHEN** a phase mode is sent for a charging charger and the pacing window stores the pause
- **THEN** no re-sync is requested before that pause is sent

#### Scenario: a re-sync is already pending
- **WHEN** a phase mode is sent for an idle charger 14 s after an accepted stop
- **THEN** the re-sync is requested 15 s after the stop

#### Scenario: hub clock behind Easee
- **WHEN** the cached observations carry timestamps ahead of the hub clock
- **THEN** the requested mode is still stamped past them and outranks them in the next report

#### Scenario: only one settable mode
- **WHEN** the grid and phase count leave a single settable mode
- **THEN** no Easee call is made, no requested mode is recorded, and the command reports success

#### Scenario: target already in effect
- **WHEN** the target internal mode equals the cached internal mode and no other mode was sent in
  the last 15 s
- **THEN** no Easee call is made and the command reports success

#### Scenario: Easee rejects the change
- **WHEN** the set-phase-mode call fails
- **THEN** the error is returned, neither the target nor the requested mode is cached, and no
  re-sync is armed
