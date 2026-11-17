## MODIFIED Requirements

### Requirement: Superseded Default Migrations
A stored `offered_current_wait_time` of exactly `15s` (the 2.8 packaged default) or `20s` (the
3.1.2 packaged default) SHALL be rewritten to `30s`, a stored SignalR `finalBackoff` of exactly
`2m` SHALL be rewritten to `10m`, and a stored `pollingInterval` of exactly `30s` (the packaged
default up to 2.6.3) SHALL be rewritten to `10m` by the migration to config version 9. Any other
value SHALL be left alone. A value deliberately chosen to match an old default is indistinguishable
from it and is rewritten too. The wait-time migration SHALL run again for installs already past it,
so a 3.1.2 install is lifted on its next boot.

#### Scenario: old packaged default
- **WHEN** `offered_current_wait_time` is `15s` or `20s`
- **THEN** it becomes `30s`

#### Scenario: user-chosen value
- **WHEN** `offered_current_wait_time` is `45s`
- **THEN** it is left unchanged

#### Scenario: SignalR final backoff
- **WHEN** the SignalR `finalBackoff` is `2m`
- **THEN** it becomes `10m`

#### Scenario: pre-2.6.4 polling interval
- **WHEN** a version 8 configuration carries `pollingInterval` `30s`
- **THEN** it becomes `10m` and the configuration reaches version 9

#### Scenario: tuned polling interval
- **WHEN** `pollingInterval` is `5m`
- **THEN** it is left unchanged
