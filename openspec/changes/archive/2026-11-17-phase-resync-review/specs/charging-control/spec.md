## MODIFIED Requirements

### Requirement: Teardown Waits Out Immediate Sends
Teardown SHALL wait for a start, stop or dynamic-current write that is being sent at once until it
reaches Easee, and SHALL cancel the pending re-sync. A command Easee answers after Teardown, a phase
mode among them, SHALL arm no re-sync.

#### Scenario: thing deleted during an immediate stop
- **WHEN** the thing is deleted while a stop sent at once is still waiting for Easee
- **THEN** Teardown returns only after the stop finished, and no re-sync fires afterwards

#### Scenario: thing deleted during a phase mode set
- **WHEN** the thing is deleted while a set-phase-mode call is still waiting for Easee
- **THEN** no re-sync fires afterwards

#### Scenario: command after Teardown
- **WHEN** a start, stop or dynamic-current write for the charger arrives after Teardown
- **THEN** it never reaches Easee and arms no re-sync
