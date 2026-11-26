## 1. Tests
- [x] 1.1 Suite: 3 A pauses once and gets exactly one session report
- [x] 1.2 Suite: a failing pause calls Easee once and gets one error report (failed before: answered with a session report)

## 2. Removal
- [x] 2.1 Delete `routeOfferedPause`, `divertOfferedPause` and their tests
- [x] 2.2 Narrow the routing `Controller` interface to `Bind`/`Unbind`
- [x] 2.3 Controller guards kept: `clamp`, `offeredPause` and the user-stop recheck in `dispatchPair` also serve the observation and timer paths; comments no longer cite the divert
- [x] 2.4 Archive issue 56
