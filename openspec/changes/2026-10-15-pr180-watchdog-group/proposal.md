# PR #180 review fixes, round 2

## Why
- A `TotalPower` of zero kept the charging watchdog quiet although zero power does not confirm charging.
- The service took its group from easee's primary group, which a failed `usermod` leaves as the legacy one.

## What Changes
- Only a `TotalPower` above zero counts as watchdog activity.
- The systemd unit sets `Group=futurehome`.

## Impact
- Affected specs: `charging-control`
- Affected code: `src/internal/signalr/manager.go`, `package/debian/usr/lib/systemd/system/easee.service`
