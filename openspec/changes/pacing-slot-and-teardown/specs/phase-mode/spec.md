# Phase Mode

## MODIFIED Requirements

### Requirement: Session Restart To Apply A Mode
Because the charger applies a new phase mode only at a session boundary, an in-progress session SHALL
be bounced after the mode is set: a pause followed by a resume, both through the per-charger command
channel. The restart SHALL be skipped when the charger state is not charging. A failure to read the
state, or a failure to pause, SHALL be logged as a warning and treated as success — the mode is
stored and takes effect on the next session anyway. The pause and the resume SHALL be dispatched as
one ordered pair, so a mode change never leaves the charger paused: whether the pause goes out at
once or is stored, the resume follows it one `offered_current_wait_time` later. The command SHALL
report success once the pair is accepted. Because the pair is stored as a unit, the resume can no
longer fail on its own after the pause has been accepted - the failure the `charger was left
stopped` error reported no longer arises. A resume Easee refuses or the charger never echoes back is
logged only. The
resume SHALL restore the session's own current — the cached requested offered current, falling back
to the cached offered current the charger itself reports, and to the cached max current when both
are zero or less — read before the pause, since the session-finished observation clears it
asynchronously. When none of the three is known the resume SHALL NOT be attempted and the failure
SHALL be reported, rather than issuing a zero-current resume that leaves the charger paused. It
SHALL NOT be issued as a normal-mode start, which would floor the current to
`initial_charging_current` and silently raise a slow session; the session's mode is recorded nowhere,
so it cannot be reconstructed.

#### Scenario: charger idle
- **WHEN** the charger state is not charging
- **THEN** no stop or start is issued

#### Scenario: a session is bounced
- **WHEN** the pause is sent for a charging charger
- **THEN** the resume is stored and sent at the channel's deadline, and the command reports success

#### Scenario: pause inside the window
- **WHEN** a dynamic-current write went out within `offered_current_wait_time` before the pause
- **THEN** the pause is stored ahead of the resume, is sent at the deadline, and the resume follows
  one wait time later, leaving the charger charging on the new mode

#### Scenario: the restart is preempted
- **WHEN** a dynamic-current write arrives while the restart is stored
- **THEN** both halves are discarded, only that write is sent, and the mode takes effect at the next
  session

#### Scenario: a slow session is bounced
- **WHEN** a session running below `initial_charging_current` is bounced to apply a mode
- **THEN** it resumes at the current it was running, not at `initial_charging_current`

#### Scenario: pause fails
- **WHEN** stopping the session fails
- **THEN** a warning is logged and the command reports success

#### Scenario: resume fails
- **WHEN** Easee refuses the resume when the deadline fires
- **THEN** the refusal is logged at error level; the command had already reported success

#### Scenario: resume is never confirmed
- **WHEN** the deferred resume is sent but no observation echoes the current back in time
- **THEN** a warning is logged; the command had already reported success

#### Scenario: no resume current is known
- **WHEN** the requested offered current, the offered current and the max current are all zero
- **THEN** no resume is attempted and the command fails

#### Scenario: charger state unknown
- **WHEN** the state report fails
- **THEN** a warning is logged and the command reports success
