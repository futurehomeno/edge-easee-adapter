## ADDED Requirements

### Requirement: Login Request
`Authenticator.Login` SHALL refuse a username or password that is empty or only whitespace with
`ErrEmptyCredentials`, without calling Easee, because Easee counts failed logins per account and locks the account
for about an hour, which then also rejects the user's valid logins. The login request SHALL send the username and
password with surrounding whitespace removed, as `userName` and `password`, with content type `application/*+json`.
An answer of 400, 401 or 403 SHALL be reported as `ErrInvalidCredentials`; any other status except 200 SHALL be
returned as an error naming the status code.

#### Scenario: blank password
- **WHEN** `Login` is called with the password `"   "`
- **THEN** it returns `ErrEmptyCredentials` and no request is sent

#### Scenario: Easee refuses the password
- **WHEN** the login is answered with 401
- **THEN** the error is `ErrInvalidCredentials` and names status code 401

#### Scenario: Easee fails
- **WHEN** the login is answered with 500
- **THEN** the error names status code 500 and is not `ErrInvalidCredentials`

### Requirement: Application Login Lifecycle
`cmd.auth.login` on an app that is not configured SHALL log in through the authenticator and then register the
selected chargers. When the login fails, or the chargers cannot be registered, the app SHALL stay not configured
(app health not configured, config not configured, disconnected, not authenticated) and the error SHALL be returned.
On success the app SHALL be marked running (configured, authenticated, connected) and then ping Easee, so a failed
ping leaves it authenticated but disconnected. What a failed login does to an app that is already running is open
(issue 52).

#### Scenario: successful login
- **WHEN** Easee accepts the credentials and the chargers are registered
- **THEN** the app is running, configured, authenticated and connected

#### Scenario: Easee refuses the login
- **WHEN** the authenticator returns an error
- **THEN** the app is not configured, not authenticated and disconnected, and the error is returned

#### Scenario: chargers cannot be registered
- **WHEN** the login succeeds but registering the chargers fails
- **THEN** the app is not configured, not authenticated and disconnected, and the error is returned

#### Scenario: ping fails after the login
- **WHEN** the login and the registration succeed but the ping fails
- **THEN** the app is running, configured and authenticated, and disconnected

## MODIFIED Requirements

### Requirement: Stored Password Encrypted
The credential store SHALL write the username and password to the secrets file encrypted with
AES-256-GCM under a key built into the adapter, and SHALL read back a plain-text value written by an
earlier version. A value that does not decrypt SHALL read as empty without failing the load. A load that leaves the
username empty SHALL drop the password too, so a username that did not decrypt never sends a re-login with an empty
one.

#### Scenario: Secrets file holds no plain password
- **WHEN** credentials with a username and password are saved
- **THEN** the secrets file contains neither in plain text and loading it returns both unchanged

#### Scenario: Plain-text password from an earlier version
- **WHEN** the secrets file holds a username and password in plain text
- **THEN** loading it returns both unchanged

#### Scenario: Password that does not decrypt
- **WHEN** the stored password cannot be decrypted
- **THEN** loading keeps the tokens and reads the password as empty

#### Scenario: Username that does not decrypt
- **WHEN** the stored username cannot be decrypted and the password can
- **THEN** loading keeps the tokens and reads both the username and the password as empty

#### Scenario: Plain-text secrets sealed at startup
- **WHEN** the adapter starts with a secrets file, or its backup, holding a plain-text username or password
- **THEN** it rewrites the file encrypted and removes the backup that holds the plain text
