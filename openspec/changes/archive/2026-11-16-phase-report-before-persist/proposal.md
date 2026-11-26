# Publish the phase-mode report before the leg is saved

## Why
- A new output leg is saved to the thing state inside the queued phase-mode report, before the
  report is published. A write that blocks (slow or worn flash) holds that report back, and every
  later report of the charger behind it: one sender publishes them in order. The spec already says
  the report is published when the write blocks; only a failed write met that (review of PR #213).

## What Changes
- The queued phase-mode report is published first and the leg is saved after it. The write stays
  off the observation drain and its error is still logged.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/signalr/handler.go`
