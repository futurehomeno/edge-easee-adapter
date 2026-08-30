# Cloud Authentication Specification

## Purpose
Obtain and keep alive an Easee cloud session for the hub. Covers the username/password login, the
persisted credential pair, the refresh-token exchange, the backoff and grace policy around
rejections, the auth-loss escalation, and explicit logout.

## Requirements

### Requirement: Password Login
`Authenticator.Login` SHALL exchange a username and password at `POST /api/accounts/login` and
persist the returned credentials in the credential store. A store write failure SHALL be logged as
a warning and SHALL NOT fail the login, because the credentials remain usable in memory until the
next restart. A successful login SHALL reset the authenticator backoff.

#### Scenario: successful login
- **WHEN** `Login` is called with credentials Easee accepts
- **THEN** the access and refresh tokens are written to the credential store
- **AND** the backoff is reset
- **AND** `Login` returns no error

#### Scenario: credential store write fails
- **WHEN** Easee accepts the credentials but the store write fails
- **THEN** the failure is logged as a warning
- **AND** `Login` returns no error

#### Scenario: Easee rejects the credentials
- **WHEN** Easee rejects the username or password
- **THEN** `Login` returns the error and nothing is persisted

### Requirement: Credential Expiry Derivation
Both token expiry times SHALL be derived from the JWT `exp` claim of the token itself, falling back
to the value supplied by the caller when the claim cannot be read. Easee rotates the refresh token
on every exchange and its response carries no refresh expiry, so the local "refresh token expired"
check depends on this derivation.

#### Scenario: expiry read from the JWT
- **WHEN** a token whose `exp` claim is readable is stored
- **THEN** the stored expiry is the claim value, not the caller-supplied fallback

#### Scenario: unreadable JWT
- **WHEN** the `exp` claim cannot be parsed
- **THEN** the caller-supplied fallback time is stored and the failure is logged at debug level

### Requirement: Access Token Provision
`AccessToken` SHALL return a valid access token, refreshing it through the framework authenticator
when it has expired. When the framework reports `ErrRefreshSuspended` or `ErrRefreshDeferred` the
authenticator SHALL translate both into `ErrRefreshBackoff`, so callers log them at debug level
rather than warning on every request for the whole grace window.

#### Scenario: token still valid
- **WHEN** `AccessToken` is called and the stored access token has not expired
- **THEN** the stored token is returned without a network call

#### Scenario: refresh suspended by backoff
- **WHEN** the framework returns `ErrRefreshSuspended` or `ErrRefreshDeferred`
- **THEN** `AccessToken` returns `ErrRefreshBackoff`

#### Scenario: not logged in
- **WHEN** `AccessToken` is called with an empty credential store
- **THEN** an error identified by `ErrNotLoggedIn` is returned

### Requirement: Refresh Token Exchange
The token exchange SHALL send the expired access token alongside the refresh token to
`POST /api/accounts/refresh_token`. The access token SHALL be looked up by the refresh token the
framework chose rather than read off the store, because Easee rejects a pair drawn from two
different sessions and a login landing mid-exchange would otherwise pair two sessions.

#### Scenario: exchange uses the matching pair
- **WHEN** the framework asks to exchange a refresh token that was rotated but refused by the store
- **THEN** the access token issued alongside that same refresh token is sent with it

#### Scenario: write-through guards a stale rotation
- **WHEN** a refresh completes after a logout or a new login replaced the session
- **THEN** the store write is conditioned on the refresh token the exchange started from and the
  superseded session is not restored

### Requirement: Rejection Grace Before Auth Loss
The authenticator SHALL be configured with a stateful backoff and an unauthorized grace period
(`auth_max_unauthorized`, default 2h). A streak of rejections SHALL have to outlive the grace before
auth loss is concluded, because Easee has historically returned transient 401s on a still-valid
refresh token.

#### Scenario: transient rejection inside the grace
- **WHEN** a refresh is rejected but the streak is younger than the grace period
- **THEN** auth loss is not declared and the refresh is retried after the backoff

#### Scenario: rejection streak outlives the grace
- **WHEN** rejections continue past the grace period
- **THEN** the framework clears the credentials and invokes the auth-loss handler

### Requirement: Auth Loss Escalation
On auth loss the adapter SHALL send the `easee_status_offline` push notification and publish
`cmd.auth.logout` to `pt:j1/mt:cmd/rt:ad/rn:easee/ad:1`. It SHALL additionally run a local logout
fallback on its own goroutine, whether or not the publish succeeded, because the routed handler
takes a try-lock that discards rather than queues a concurrent command. The fallback SHALL compare
the credentials against a snapshot taken before the publishes and skip itself when a new session has
replaced the one that triggered the loss.

#### Scenario: auth loss with a reachable broker
- **WHEN** the auth-loss handler runs
- **THEN** a push notification is sent, `cmd.auth.logout` is published, and the fallback runs

#### Scenario: re-login lands during the escalation
- **WHEN** a fresh login replaces the credentials while the publishes are in flight
- **THEN** the fallback observes the changed credentials and skips the local logout

#### Scenario: notification or publish fails
- **WHEN** either the push notification or the MQTT publish returns an error
- **THEN** the error is logged and the remaining steps still run

### Requirement: Explicit Logout
`Authenticator.Logout` SHALL clear the stored credentials. The application-level logout SHALL first
close the SignalR client, logging a close failure as a warning without aborting, then clear the
credentials. On success it SHALL mark the app not configured; on failure it SHALL set app health to
error, auth state to not-authenticated and config state to not-configured, and return the error.

#### Scenario: successful logout
- **WHEN** `cmd.auth.logout` is handled and credential clearing succeeds
- **THEN** the SignalR client is closed and the app is marked not configured

#### Scenario: credential clearing fails
- **WHEN** clearing the credentials returns an error
- **THEN** app health is error, auth state is not-authenticated, config state is not-configured
- **AND** the error is returned

### Requirement: Credential Persistence
Credentials SHALL be held in a dedicated secrets store separate from the public configuration, so
that resetting the configuration cannot leave the Easee tokens on the hub. The stored model SHALL
carry the access token, the refresh token and both derived expiry times.

#### Scenario: uninstall clears both stores
- **WHEN** the app is uninstalled
- **THEN** the configuration reset and the credential clear both run, each reporting its own error

#### Scenario: empty credentials mean unconfigured
- **WHEN** the adapter initializes and the credential store is empty
- **THEN** the app is marked not configured and initialization returns without contacting Easee

### Requirement: Periodic Token Refresh
A background task SHALL refresh the token on `token_refresh_interval` (default 30m), jittered by up
to 5000ms when the interval exceeds one second so that hubs do not refresh in lockstep. The task
SHALL be skipped while the auth state is not-authenticated.

#### Scenario: refresh runs while authenticated
- **WHEN** the refresh task fires and the auth state is not `NOT_AUTHENTICATED`
- **THEN** the adapter pings Easee to exercise and refresh the token

#### Scenario: refresh skipped while logged out
- **WHEN** the refresh task fires and the auth state is `NOT_AUTHENTICATED`
- **THEN** nothing is called

### Requirement: Connection State From Ping
`RefreshToken` SHALL ping Easee and set the lifecycle connection state from the outcome. A failure
SHALL set the state to disconnected; `ErrNotLoggedIn` and `ErrRefreshBackoff` SHALL be logged at
debug level and any other failure at warning level. A success following a disconnected state SHALL
log the reconnection and set the state to connected.

#### Scenario: ping succeeds after an outage
- **WHEN** the ping succeeds and the previous state was disconnected
- **THEN** a reconnection is logged and the connection state becomes connected

#### Scenario: ping fails while backing off
- **WHEN** the ping fails with `ErrRefreshBackoff`
- **THEN** the reason is logged at debug level and the connection state becomes disconnected
