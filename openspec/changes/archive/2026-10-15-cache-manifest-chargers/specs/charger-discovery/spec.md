## ADDED Requirements

### Requirement: Manifest Charger List Cache
The manifest device selector SHALL reuse a successfully fetched charger list for 10 seconds, SHALL
not cache a failed fetch, and SHALL drop the cached list on login and logout. Other charger list
reads (login, configure, re-seed) SHALL always fetch fresh.

#### Scenario: back-to-back manifest requests
- **WHEN** the manifest is requested twice within 10 seconds while authenticated
- **THEN** the chargers are fetched from Easee once

#### Scenario: cache expired, failed or account changed
- **WHEN** the previous fetch failed, is older than 10 seconds, or a login or logout happened since
- **THEN** the next manifest request fetches the chargers again
