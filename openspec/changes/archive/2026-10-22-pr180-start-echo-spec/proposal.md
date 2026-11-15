# PR #180 review fixes, round 4

## Why
- The `2026-10-15-pacing-slot-and-teardown` delta for Start Charging was written from a copy that predates
  `2026-10-06-start-resume-wait`, so archiving it restored the removed `start accepted, but the charger did not
  resume` error to the canonical spec. The code and its test follow the 10-06 contract.

## What Changes
- Start Charging again specifies that an accepted start succeeds when the echo is missed, logging the miss at info,
  and keeps the 32A fallback the 10-15 change added.

## Impact
- Affected specs: `charging-control`
- Affected code: none (`TestController_StartChargepointCharging_UnconfirmedStartSucceeds` already covers it)
