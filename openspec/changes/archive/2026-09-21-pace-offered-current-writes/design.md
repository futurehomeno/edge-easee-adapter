# Design — pace-offered-current-writes

## The channel

```go
type chargerCommand struct{ stop bool; current int }

type controller struct {
    // ...
    pace    sync.Mutex
    sentAt  time.Time       // last attempt, stop included
    pending *chargerCommand // the single slot
}
```

`dispatch(cmd) (sent bool, err error)`: under `pace`, if the slot is empty and
`clock.Since(sentAt) >= offered_current_wait_time`, stamp `sentAt` and send outside the lock.
Otherwise store `cmd` in the slot. The one-shot `clock.AfterFunc` is armed only on the nil to
non-nil transition, for `sentAt + offered_current_wait_time - now`; a later command only
replaces the payload. `sendPending` takes the slot and stamps under `pace`, sends outside it, waits
for the echo for a warning only, and logs a refused send at error level. A nil slot is a no-op.

## Decisions

- **Stamp on attempt, not on success.** A call that times out may still have reached Easee;
  stamping only on 202 would let the next write go out inside the window.
- **One window, not two.** The first cut held a stored command for a separate
  `offered_current_deferral_time` (45 s). It had nothing to add: a stored command may go out the
  moment the channel is quiet, which is `lastSentAt + offered_current_wait_time` by definition.
  The single window is 30 s; installs still on 15 s or 20 s are migrated, since the packaged
  default is persisted at first boot and a new fallback alone would change nothing in the field.
- **Timer armed once, deadline never moves.** `lastSentAt + wait time` is fixed by the send that
  opened the window; without this a 5 s cadence would push the deadline forever.
- **A stored command does not survive a restart.** `sentAt` is zero after boot, so the first
  command goes out immediately; whatever was stored is gone. The energy guard resends on its own
  cadence, and a lost stop is the same as a stop that failed - already accepted by the spec.
- **A pause inside the window is replaced by the resume.** `restartForPhaseMode` stops then
  resumes; when the stop itself is stored, the resume overwrites it. The session is not bounced
  and the mode applies at the next session boundary - identical to today's "pause fails" outcome.
- **Latest wins, even across kinds.** A `set_current` replaces a stored stop and a stop replaces a
  stored `set_current`. The later intent is the user's current intent.
- **The deferred outcome is log-only.** The FIMP command that stored the command has already
  answered; the echo, or its absence, shows up in the SignalR-driven reports.
- **Mock clock in tests.** `clock.Mock(t0)` + `mock.Add(d)`; a fired deadline runs on a goroutine
  and is awaited through a `done` channel signalled from the last mock call of the path. These
  tests cannot be `t.Parallel()` - the clock is global.
- **`force` shrinks to "bypass the cache dedup".** With the client throttle gone it has nothing
  else to bypass.
