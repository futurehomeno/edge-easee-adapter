# Cloud Authentication

## MODIFIED Requirements

### Requirement: Explicit Logout
`Authenticator.Logout` SHALL clear the stored credentials. The application-level logout SHALL first
close the SignalR client, logging a close failure as a warning without aborting, then clear the
credentials. It SHALL mark the app not configured only once the credential store is actually empty:
`Initialize` treats a non-empty store as a live session and re-authenticates from it, so a logout
that left tokens on disk while reporting itself successful is silently undone by the next hub
restart. When the clear fails but the store is empty regardless, the logout SHALL stand. When
credentials remain readable, the app SHALL set app health to error, auth state to
not-authenticated, config state to not-configured and the connection state to disconnected — the
reporting tasks are gated on the connection state, so leaving it connected keeps them polling for a
session the app has declared dead — and return the error.

#### Scenario: successful logout
- **WHEN** `cmd.auth.logout` is handled and credential clearing succeeds
- **THEN** the SignalR client is closed and the app is marked not configured

#### Scenario: credential clearing fails
- **WHEN** clearing the credentials returns an error and credentials are still readable
- **THEN** app health is error, auth state is not-authenticated, config state is not-configured
- **AND** the error is returned

#### Scenario: clearing reports an error but leaves no credentials
- **WHEN** clearing the credentials returns an error but the store is empty afterwards
- **THEN** the app is marked not configured, because a restart would find nothing to resume from
