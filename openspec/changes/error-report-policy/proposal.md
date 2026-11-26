# Error report policy

## Why
Cliffhanger's error report policy (futurehomeno/cliffhanger#292) answers every failed command with a
code and a plain sentence, and a failure found after the command was accepted with an
`evt.error.report` carrying the command's `corid`. Easee defers writes inside its pacing window: when
Easee refused such a write the command had already been answered as accepted and the refusal reached
only the log. HTTP failures were logged twice (by the HTTP client and by the router), the error text
carried URLs and response bodies, and a comment and the spec said cliffhanger rejects an offered
current under 6 A, which it does not.

## What Changes
- cliffhanger pinned to `v1.5.6-0.20261125115807-4a5e15820840` (to be re-pinned to the release).
- HTTP failures are classified once in the HTTP client into `unauthorized`, `not_found`,
  `unavailable`, `rate_limited` (429) and `device_refused`, with the URL and body kept in the cause for the log; the
  client's own log line goes.
- A lost stream answers report requests with `unavailable`.
- Every chargepoint command binds its charger's controller from the first route to the last, so a
  deferred write keeps its command; report requests are not bound and are answered at once; a refused deferred write publishes `router.NewErrorReport`.
- The unmapped phase mode is returned, not also logged.
- The wrong "cliffhanger rejects under 6 A" statements in code and spec are corrected.
- The app manifest already declares `out evt.error.report string` (checked by its test): no change.

## Impact
- Capabilities: `charging-control`.
- Code: `internal/api/http.go`, `internal/easee/controller.go`, `internal/easee/thing.go`,
  `internal/routing/router.go`, `cmd/factory.go`, `internal/config/config.go`.
