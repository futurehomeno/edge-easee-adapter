# pace-offered-current-writes

## Why

Customer log 2026-09-20 (core-energy-guard 4.4.3, easee 3.1.2): Easee ignores a second
offered-current write that reaches the charger within ~20 s of the previous one, even though the
cloud answers 202 with `wasAccepted`. `restartForPhaseMode` sends `StopCharging` and the resume
`UpdateDynamicCurrent(resume, force=true)` 10 ms apart. In 17 of 17 phase switches the charger was
parked at 0 A in `await_start` while the adapter reported success: `WaitForOfferedCurrent`
returned true on the stale, still-equal cache. The energy guard never re-arms such a charger, so
the car went eight nights without charging.

The client-side throttle (`offered_current_wait_time`, 20 s) was meant to keep writes apart, but
`force` bypasses it on exactly the path that needs it, and the throttle answers a refused write
with an error the caller cannot act on. Refusing is the wrong primitive: the write has to reach
the charger, only later.

## What Changes

- **One per-charger command channel** in the controller for every dynamic-current write and
  pause. It records when the last command was sent (a stop included). A command that arrives on
  a quiet channel - at least `offered_current_wait_time` since the last send - goes out at once.
  Any other command is stored in a single slot and sent at `lastSentAt + offered_current_deferral_time`
  (new setting, 45 s). A newer command replaces the stored payload and never moves the deadline,
  so a stream of writes every 5 s yields exactly one send per 45 s, carrying the latest value.
- **A deferred command succeeds at once.** The FIMP command returns nil; cliffhanger's forced
  report carries the value in effect, and the SignalR echo after the deferred send is the
  confirmation. The deferred send runs on its own goroutine, never under the per-thing service
  lock, awaits the echo only to log it, and logs a refused send at error level.
- **The phase-mode resume is deferred, not fired.** The pause goes out, the resume is stored and
  sent 45 s later, and the command reports success once the pause is accepted. A pause that
  itself lands inside the window is replaced by the resume: no bounce, the mode applies at the
  next session boundary - the same outcome the spec already accepts for a failed pause.
- **The HTTP client's throttle and the `force` argument are removed.** `NewHTTPClient` no longer
  takes the config service; `UpdateDynamicCurrent` loses its third argument in both interfaces.
  `force` on the controller's internal setter now means only "bypass the cache dedup".
- **`offered_current_deferral_time`** (default 45 s) joins the packaged configuration.

## Superseded

[PR #151](https://github.com/futurehomeno/edge-easee-adapter/pull/151) drops the client throttle
in the opposite direction - every write goes out. It is superseded by this change.

## Out of scope

The energy guard re-arming a charger that went `charging -> ready_to_charge` right after an
EG-issued phase switch is a separate follow-up in core-energy-guard.
