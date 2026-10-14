## MODIFIED Requirements

### Requirement: Re-login With Stored Password
Easee ends a session 60 days after the password login, whatever refreshes happen in between. While
a password is stored, the token exchange SHALL log in again with the stored username and password
instead of refreshing when the refresh token expires within 25 hours or has expired, and SHALL do
the same when Easee rejects a refresh as unauthorized. The new session SHALL be written through the
same guarded write as a refresh, so a logout or login landing during the re-login wins. A login
Easee rejects with 400, 401 or 403 SHALL forget the stored password and report the exchange as
unauthorized, so the app ends the session as it did before and never repeats a rejected login.
Any other failure SHALL keep the password, leaving the next attempt to the backoff.

#### Scenario: session ends
- **WHEN** a token is needed after the access and refresh tokens expired and a password is stored
- **THEN** the adapter logs in with the stored username and password and stores the new tokens
- **AND** no auth loss is reported

#### Scenario: session about to end
- **WHEN** the access token is due for refresh and the refresh token expires within 25 hours
- **THEN** the adapter logs in with the stored password instead of refreshing

#### Scenario: login fails before the session ends
- **WHEN** that login fails with a transport error or a 5xx while the refresh token is still valid
- **THEN** the adapter refreshes instead

#### Scenario: refresh rejected
- **WHEN** Easee rejects a refresh as unauthorized and a password is stored
- **THEN** the adapter logs in with the stored password

#### Scenario: password rejected
- **WHEN** the re-login is rejected with 400, 401 or 403
- **THEN** the stored password is forgotten and the exchange is reported as unauthorized
- **AND** no later token request logs in with it

#### Scenario: transient failure
- **WHEN** the re-login fails with a transport error or a 5xx
- **THEN** the password is kept and a later token request logs in again

#### Scenario: logout during the re-login
- **WHEN** a logout or another login lands while the re-login is in flight
- **THEN** the re-login's tokens are not stored
