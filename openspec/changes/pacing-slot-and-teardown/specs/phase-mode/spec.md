# Phase Mode

## MODIFIED Requirements

### Requirement: Session Restart To Apply A Mode
Because the charger applies a new phase mode only at a session boundary, an in-progress session SHALL
be bounced after the mode is set: a pause followed by a resume, both through the per-charger command
channel. The restart SHALL be skipped when the charger state is not charging. A failure to read the
state, or a failure to pause, SHALL be logged as a warning and treated as success — the mode is
stored and takes effect on the next session anyway. When the pause is sent at once — the ordinary
case, since a mode change rarely follows a current write inside the window — the resume SHALL be
stored and SHALL go out at the channel's deadline, and the command SHALL report success once the
pause is accepted. When the pause is itself stored rather than sent, it SHALL hold the slot: the
resume SHALL be dropped, the session SHALL end at the deadline, and the mode SHALL take effect at
the next session — the outcome the spec already accepts for a failed pause, reached by a different
route. A stop is a safety command and is never cancelled by a later current write, so the resume
cannot displace it. A resume Easee refuses or the charger never echoes back is logged only, and a resume the channel
cannot accept SHALL be returned as `phase mode set to <target>, but the charger was left stopped`. The
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

#### Scenario: the pause is itself deferred
- **WHEN** a mode change lands within `offered_current_wait_time` of a current write, so the pause is
  stored rather than sent
- **THEN** the pause holds the slot, the resume is dropped, and the session ends at the deadline with
  the mode taking effect at the next session

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

#### Scenario: a resume the channel cannot accept
- **WHEN** the resume cannot be dispatched at all
- **THEN** the command fails with `phase mode set to <target>, but the charger was left stopped`

#### Scenario: no resume current is known
- **WHEN** the requested offered current, the offered current and the max current are all zero
- **THEN** no resume is attempted and the command fails

#### Scenario: charger state unknown
- **WHEN** the state report fails
- **THEN** a warning is logged and the command reports success
