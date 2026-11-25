## ADDED Requirements

### Requirement: Easee HTTP Requests
Every call to the Easee REST API other than the login and the token refresh SHALL carry the header
`Authorization: Bearer <access token>` and go to the configured `easee_base_url`. The commands SHALL be sent as:
- dynamic (offered) current: `POST /api/chargers/{id}/settings` with `{"dynamicChargerCurrent": <A>}`;
- max current: `POST /api/chargers/{id}/settings` with `{"maxChargerCurrent": <A>}`;
- stop: `POST /api/chargers/{id}/commands/pause_charging` with no body and no content type;
- cable lock: `POST /api/chargers/{id}/commands/lock_state` with `{"state": <bool>}`;
- phase mode: `POST /api/chargers/{id}/commands/set_phase_mode` with `{"phaseMode": <1|2|3>}`.

A command with a body SHALL carry the content type `application/*+json`. The reads SHALL be `GET /api/chargers`,
`GET /api/chargers/{id}/config`, `GET /api/chargers/{id}/site`,
`GET /api/chargers/{id}/details?alwaysGetChargerAccessLevel=false` and the health check `GET /health`; each SHALL
succeed only on 200, and any other status SHALL be logged with the response body and returned as an error naming
the status code. A read whose body does not decode SHALL fail.

#### Scenario: offered current written
- **WHEN** 10 A is written as the dynamic current of charger `XX12345`
- **THEN** `POST /api/chargers/XX12345/settings` is sent with `{"dynamicChargerCurrent":10}` and the bearer token

#### Scenario: stop sent
- **WHEN** a stop is sent for charger `XX12345`
- **THEN** `POST /api/chargers/XX12345/commands/pause_charging` is sent with no body

#### Scenario: read answered with an error
- **WHEN** `GET /api/chargers/XX12345/config` is answered with 500
- **THEN** the body is logged and an error naming status code 500 is returned
