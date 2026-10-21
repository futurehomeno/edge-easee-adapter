# auth-loss-telemetry

## Why

When Easee rejects the refresh token, the auth-loss handler logs the app out but never marks the
lifecycle auth state `LOST`. cliffhanger's auth-loss watcher only reports an `AUTHENTICATED -> LOST`
transition (and disarms on `NOT_AUTHENTICATED`), so no `auth`/`logged_out` telemetry event and no
app state report is sent when the hub gets logged out.

## What Changes

- The auth-loss handler sets the lifecycle to disconnected + `LOST` with the loss reason before the
  logout, so cliffhanger's watcher emits the `auth`/`logged_out` event (same as the Zaptec adapter).

## Impact

- `src/internal/api/authenticator.go`, `src/cmd/factory.go`
