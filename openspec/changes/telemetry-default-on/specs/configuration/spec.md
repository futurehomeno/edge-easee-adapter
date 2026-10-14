## ADDED Requirements

### Requirement: Telemetry Enabled By Default
A configuration without a telemetry block SHALL get cliffhanger's default: enabled, with the 30-day
validity starting at startup. The migration to config version 8 SHALL switch a stored block that is
disabled with a zero `enabled_at` (the seed written by earlier versions) on, keeping its validity and
suppression, with its window starting at startup, and SHALL run only once.

#### Scenario: fresh install
- **WHEN** the adapter starts with no telemetry block
- **THEN** telemetry is enabled

#### Scenario: old disabled seed
- **WHEN** a version 7 configuration carries a disabled telemetry block with a zero `enabled_at`
- **THEN** the migration enables it and the configuration reaches version 8

#### Scenario: already migrated
- **WHEN** a version 8 configuration carries a disabled telemetry block
- **THEN** it is left disabled
