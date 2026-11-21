# Design

## Decision
The controller keeps the last phase target Easee accepted and the hub-clock time of that send. A
target equal to the cached internal mode is skipped as before, unless the kept target differs from it
and was sent less than `resyncDelay` (15 s) ago. The cache has then moved off the last send - an older
echo (issue 11), a replay (issue 10) or a change made elsewhere - while the charger may still apply
that send, so the target takes the normal path: sent, cached, recorded, then bounced or re-synced.

- The cache stays the only source of the internal mode, for the comparison and for the report. The
  kept send only turns a skip into a send: it is never written into the cache, never compared with
  an observation, and never makes a request skip.
- It lapses with the re-sync window: by then the re-sync the send armed, or the earlier one it kept,
  has replayed the charger's real mode, so a request equal to the cache is a no-op again. Both are
  measured on the hub clock, so no Easee timestamp is involved.
- A bounced session re-syncs from its stop and its resume, which the pacing window can hold back past
  15 s; the window is not stretched to them. An echo slower than 15 s keeps the gap issue 10 leaves
  open.
- Locking: cliffhanger's chargepoint service (v1.5.3, `service.SetPhaseMode`) holds its lock around
  `SetChargepointPhaseMode`, its only caller, so the fields sit next to `clamp` under the same rule.

## Why this is not the rejected alternative
`phase-set-before-echo` rejected comparing against a last-sent target held in the controller,
because it would be a second source of the internal mode with its own timestamp rule and lock. That
design would have replaced the cache in the comparison. Here the cache still decides; the kept send
can only veto a skip for 15 s, needs no timestamp rule against observations, and is serialized by the
service lock already held.

## Accepted cost
A mode changed elsewhere (the Easee app) within 15 s of a send and then requested is sent again: one
more Easee call, and on a charging charger one more session bounce.
