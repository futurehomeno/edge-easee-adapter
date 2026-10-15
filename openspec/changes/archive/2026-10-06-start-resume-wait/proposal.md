# start-resume-wait

## Why

The 3.1.4 beta reported `start accepted, but the charger did not resume at 13A` on a start the
charger did act on: the echo arrived 13s later and the session went to `charging`. Since #124,
`cmd.charge.start` fails unless the charger echoes the offered current within
`current_wait_duration` (3s). A paused charger is often slower than that. Across 29 beta sites, 11 of
316 resumes from 0A were echoed after 3s (4–16s), so about 1 start in 30 is reported to energy guard
as failed while the car charges.

A longer wait is not the fix. The wait runs under cliffhanger's chargepoint service lock, so
it would hold back a stop or a lower current from energy guard. It would also stall the report
sender and occupy a router worker for the whole wait.

## What Changes

- A start Easee accepts succeeds even when the echo does not arrive within `current_wait_duration`.
  The miss is logged at info, and the state report shows whether the charger resumed.
- The wait and its 3s bound are unchanged, so the service lock is held no longer than before.

## Impact

- Affected specs: `charging-control`
- Affected code: `src/internal/easee/controller.go`
- A start that Easee accepted but the charger never acts on is no longer reported as an error. The
  state report still shows the charger paused.
