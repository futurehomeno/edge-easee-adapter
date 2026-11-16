## MODIFIED Requirements

### Requirement: Supported Observations
The adapter SHALL handle these observation IDs: detected power grid type, phase mode, max charger
current, dynamic charger current, charger operating state, output phase, total power, lifetime
energy, energy session, cable rating, error code, in-current T2/T3/T4/T5, cloud connected, cable
locked, lock cable permanently, charging session start and charging session stop.

#### Scenario: dispatch table coverage
- **WHEN** an observation with any of these IDs arrives for a registered charger
- **THEN** its dedicated handler runs

#### Scenario: in-current T2 passes the receiver
- **WHEN** Easee streams in-current T2 (observation 182)
- **THEN** the receiver buffers it for the handler instead of dropping it as unsupported
