## ADDED Requirements

### Requirement: Sub-Minute Polling Interval Migration
The migration to config version 9 SHALL rewrite a stored `pollingInterval` shorter than one minute
to `20m`, the `30s` packaged default up to 2.5.10 included. Each poll of an idle charger fetches its
config and site info over REST. An interval of one minute or longer SHALL be left alone, and so SHALL
an unset or unparsable one, which the getter already reads as its `10m` fallback.

#### Scenario: 2.5 polling interval
- **WHEN** a version 8 configuration carries `pollingInterval` `30s`
- **THEN** it becomes `20m` and the configuration reaches version 9

#### Scenario: one minute or longer
- **WHEN** `pollingInterval` is `1m` or `5m`
- **THEN** it is left unchanged

#### Scenario: unset or unparsable
- **WHEN** `pollingInterval` is empty or `soon`
- **THEN** it is left unchanged
