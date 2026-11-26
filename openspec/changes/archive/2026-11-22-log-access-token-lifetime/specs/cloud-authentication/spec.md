## ADDED Requirements

### Requirement: Access Token Lifetime Logging
Every token response Easee accepts, from the password login, the re-login with the stored password
and the refresh exchange, SHALL be logged at info level as
`[auth] Access token expiresIn=<n>s expiresAt=<time>`. `<n>` is the `expiresIn` the response
carried. `<time>` is the access token expiry derived as in Credential Expiry Derivation - the JWT
`exp` claim, falling back to now plus `expiresIn` - in RFC 3339 local time, the zone of the log
timestamps. The line exists so field logs show the lifetime Easee actually issues; no refresh
timing depends on it. It is info, not debug, because hubs ship with `log_level: info`: beta logs
then carry it without a level change, at about one line per token lifetime.

#### Scenario: login logs the lifetime
- **WHEN** a login returns `expiresIn` 3600 and an access token whose `exp` claim is T
- **THEN** an info line reads `[auth] Access token expiresIn=3600s expiresAt=<T in RFC 3339 local time>`

#### Scenario: refresh logs the lifetime
- **WHEN** a token refresh returns `expiresIn` 3600 and an access token whose `exp` claim is T
- **THEN** an info line reads `[auth] Access token expiresIn=3600s expiresAt=<T in RFC 3339 local time>`
