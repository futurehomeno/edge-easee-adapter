## MODIFIED Requirements

### Requirement: Service User Group
The package SHALL give the `easee` user the primary group `futurehome`, also when the user already
exists from the thingsplex layout, so that the files the adapter and the migration write carry the
`futurehome` group. After the state migration postinst SHALL re-own the data tree to
`easee:futurehome`.

#### Scenario: upgraded from the thingsplex layout
- **WHEN** postinst configures a hub where the `easee` user already exists
- **THEN** it sets the user's primary group to `futurehome`

#### Scenario: group change fails
- **WHEN** postinst cannot change the existing user's primary group
- **THEN** it warns on stderr and the install continues

#### Scenario: migrated files
- **WHEN** postinst has run the state migration
- **THEN** it re-owns the data tree to `easee:futurehome`, whatever group the migration wrote with

#### Scenario: re-own fails
- **WHEN** postinst cannot re-own the migrated data tree
- **THEN** it warns on stderr and the install continues

#### Scenario: fresh install
- **WHEN** postinst configures a hub without the `easee` user
- **THEN** it creates the user with primary group `futurehome`
