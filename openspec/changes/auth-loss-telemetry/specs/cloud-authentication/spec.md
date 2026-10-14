## MODIFIED Requirements

### Requirement: Auth Loss Escalation
On auth loss the adapter SHALL send the `easee_status_offline` push notification and publish
`cmd.auth.logout` to `pt:j1/mt:cmd/rt:ad/rn:easee/ad:1`. Both SHALL happen on their own goroutine
rather than in the callback, which the framework invokes holding the authenticator lock: a publish
blocks on a saturated or unreachable broker for the transport write timeout, and holding that lock
stalls every access-token caller. Before them, on that same goroutine, it SHALL set the lifecycle
to disconnected and auth state `LOST` with the loss reason, so cliffhanger's auth-loss watcher
emits the `auth`/`logged_out` telemetry event and the app state report. It SHALL additionally run a
local logout fallback on that same goroutine, whether or not the publish succeeded, because the
routed handler takes a try-lock that discards rather than queues a concurrent command. The
escalation SHALL compare the credentials against a snapshot taken when the loss was detected —
before the publishes, which are no longer ordered ahead of it — and skip itself when a new session
has replaced the one that triggered the loss.

#### Scenario: auth loss with a reachable broker
- **WHEN** the auth-loss handler runs
- **THEN** the auth state becomes `LOST` with the reason, a push notification is sent,
  `cmd.auth.logout` is published, and the fallback runs

#### Scenario: auth loss is reported to telemetry
- **WHEN** an authenticated session is lost
- **THEN** the lifecycle reaches `LOST` before the logout marks it not-authenticated, so the
  `auth`/`logged_out` telemetry event is emitted

#### Scenario: broker is unreachable
- **WHEN** the auth-loss handler runs while a publish would block
- **THEN** the handler returns without waiting for it, so the authenticator lock is not held

#### Scenario: re-login lands during the escalation
- **WHEN** a fresh login replaces the credentials while the publishes are in flight
- **THEN** the fallback observes the changed credentials and skips the local logout

#### Scenario: notification or publish fails
- **WHEN** either the push notification or the MQTT publish returns an error
- **THEN** the error is logged and the remaining steps still run
