# packaging Specification

## Purpose
How the Debian package sets up the service user and its files.
## Requirements
### Requirement: Service User Group
The package SHALL give the `easee` user the primary group `futurehome`, also when the user already
exists from the thingsplex layout, so that the files the adapter and the migration write carry the
`futurehome` group.

#### Scenario: upgraded from the thingsplex layout
- **WHEN** postinst configures a hub where the `easee` user already exists
- **THEN** it sets the user's primary group to `futurehome`

#### Scenario: fresh install
- **WHEN** postinst configures a hub without the `easee` user
- **THEN** it creates the user with primary group `futurehome`

