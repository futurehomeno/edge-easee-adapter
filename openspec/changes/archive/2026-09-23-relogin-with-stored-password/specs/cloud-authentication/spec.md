## ADDED Requirements

### Requirement: Re-login With Stored Password
Easee ends a session 60 days after the password login, whatever refreshes happen in between. While
a password is stored, the token exchange SHALL log in again with the stored username and password
instead of refreshing when the refresh token expires within one hour or has expired, and SHALL do
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
- **WHEN** the access token is due for refresh and the refresh token expires within one hour
- **THEN** the adapter logs in with the stored password instead of refreshing

#### Scenario: refresh rejected
- **WHEN** Easee rejects a refresh as unauthorized and a password is stored
- **THEN** the adapter logs in with the stored password

#### Scenario: password rejected
- **WHEN** the re-login is rejected with 400, 401 or 403
- **THEN** the stored password is forgotten
- **AND** the next token request ends in auth loss without another login attempt

#### Scenario: transient failure
- **WHEN** the re-login fails with a transport error or a 5xx
- **THEN** the password is kept and a later token request logs in again

#### Scenario: logout during the re-login
- **WHEN** a logout or another login lands while the re-login is in flight
- **THEN** the re-login's tokens are not stored

## MODIFIED Requirements

### Requirement: Password Login
`Authenticator.Login` SHALL exchange a username and password at `POST /api/accounts/login` and
persist the returned credentials, together with the username and password it was called with, in
the credential store. A store write failure SHALL be logged as
a warning and SHALL NOT fail the login, because the credentials remain usable in memory until the
next restart. A successful login SHALL reset the authenticator backoff. A 200 response carrying no
access token SHALL be rejected as an error, on both the login and the token-refresh path.

#### Scenario: successful login
- **WHEN** `Login` is called with credentials Easee accepts
- **THEN** the access and refresh tokens, the username and the password are written to the
  credential store
- **AND** the backoff is reset
- **AND** `Login` returns no error

#### Scenario: credential store write fails
- **WHEN** Easee accepts the credentials but the store write fails
- **THEN** the failure is logged as a warning
- **AND** `Login` returns no error

#### Scenario: Easee rejects the credentials
- **WHEN** Easee rejects the username or password
- **THEN** `Login` returns the error and nothing is persisted

### Requirement: Credential Persistence
Credentials SHALL be held in a dedicated secrets store separate from the public configuration, so
that resetting the configuration cannot leave the Easee tokens on the hub. The stored model SHALL
carry the access token, the refresh token, both derived expiry times, and the username and
password of the last login. A token refresh SHALL keep the stored username and password.

#### Scenario: uninstall clears both stores
- **WHEN** the app is uninstalled
- **THEN** the configuration reset and the credential clear both run, each reporting its own error

#### Scenario: empty credentials mean unconfigured
- **WHEN** the adapter initializes and the credential store is empty
- **THEN** the app is marked not configured and initialization returns without contacting Easee

#### Scenario: refresh keeps the password
- **WHEN** a token refresh stores rotated tokens
- **THEN** the stored username and password are unchanged

### Requirement: Access Token Provision
`AccessToken` SHALL return a valid access token, refreshing it proactively through the framework
authenticator's `RefreshLead` window (default 5 minutes before expiry) rather than waiting for the
access token to actually expire. While no password is stored and the refresh token's local expiry
(`RefreshExpiresAt`) is already past and the access token is also expired, the framework SHALL clear the credentials and
return an error identified by `ErrReloginRequired` without attempting an HTTP refresh. When the
framework reports `ErrRefreshSuspended` or `ErrRefreshDeferred` the authenticator SHALL translate
both into `ErrRefreshBackoff`, so callers log them at debug level rather than warning on every
request for the whole grace window.

#### Scenario: token still valid
- **WHEN** `AccessToken` is called and the stored access token expires in more than `RefreshLead`
- **THEN** the stored token is returned without a network call

#### Scenario: refresh token locally expired
- **WHEN** `AccessToken` is called with an expired access token, `RefreshExpiresAt` in the past and
  no password stored
- **THEN** credentials are cleared, the auth-loss handler fires, and an error identified by
  `ErrReloginRequired` is returned

#### Scenario: refresh suspended by backoff
- **WHEN** the framework returns `ErrRefreshSuspended` or `ErrRefreshDeferred`
- **THEN** `AccessToken` returns `ErrRefreshBackoff`

#### Scenario: not logged in
- **WHEN** `AccessToken` is called with an empty credential store
- **THEN** an error identified by `ErrNotLoggedIn` is returned
