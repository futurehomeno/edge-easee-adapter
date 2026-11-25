# A load balancer's resume starts at 16 A, above the current it had balanced the charger to

Found 2026-11-23 comparing the hubsim sets (edge-sim-adapter `hubsim/.build/kpi`)
`eg-4.7.2-local_easee-3.4.2-local_zaptec-3.5.2-local_20261123T092223` and
`eg-4.7.1+release.4.7.0c8b43d_easee-3.4.1+release.3.4.ce97219_zaptec-3.5.1+release.3.5.5a295d3_20261123T062137`.

## What happens

Since PR 237 (3.4.2) `StartChargepointCharging` raises every start that is not slow to `initial_charging_current`
(16 A). energy-guard resumes a charger it paused with `cmd.charge.start` without a mode, so the resume no longer starts
at the current energy-guard last set (3.4.1) but at 16 A. The comment PR 237 removed named this cost.

it3-easee (IT 3-phase, 25 A fuse), `easee.log` of 3.4.2:

    17:52:26 Update dynamic current to 16.0      <- cmd.charge.start, 3.4.1 wrote 7.0 here
    17:52:31 State=charging                      <- L2 at 29.4 A
    17:52:41 cmd.current_session.set_current 7   <- energy-guard cuts 15 s after the start
    17:52:41 set_current 7, delayed 15s          <- the adapter's 30 s pace between writes
    17:52:56 Update dynamic current to 7.0

## Effect in the set

- it3-easee: breaker ok -> FAIL (17:52:51, over 110 % of the fuse for more than 20 s), time over the fuse 43 -> 136 s.
- it1-easee: one more start into a full line (18:00:11, 33.4 A for 41 s), time over the fuse 218 -> 259 s.
- With the charger's phase mode on auto the 16 A start is a 3-phase start; energy-guard's cut to 7 A then drops it to
  one phase and the loop of core-energy-guard issue 065 follows: it3-easee 7 -> 11 phase switches, it3-b1 11 -> 29,
  it3-b2 19 -> 35.
- One more `set_current` per restart (core-energy-guard issue 095).

## Reproduce

`make hubsim-run SCENARIO=it3 CHARGER=easee` with easee 3.4.1 and 3.4.2 on the same energy-guard; compare the first
`Update dynamic current` after each `cmd.charge.start`.

## Would close it

A decision between the two needs (a car that only starts at a higher current; a balancer whose resume must not exceed
its budget), for example: the floor only for a start that names a mode, or the first lower `set_current` after a start
passes the 30 s pace. Then the it3-easee run without the breaker FAIL.

## 2026-11-23: decided, closed

The 16 A start stays for every start that is not slow (the operator): some cars need 16 A to start charging, and after
a `set_current` the current only rises about 2 A/s. The it3-easee breaker FAIL is accepted on the adapter side. What
is left belongs to energy-guard: core-energy-guard issues 095 (no cut to the old current after a restart), 065 (lock
one phase so a 16 A start is not a 3-phase start) and 073 (a restart into a line without room for 16 A).

## Status 2026-11-25

The decision above was reversed by the operator on 2026-11-24 and PR #240 (change
`initial-current-session-start-only`) fixed it: only a session start (the charger not yet seen
charging in this session) is raised to `initial_charging_current`; a resume starts at the cached
requested current, the one energy-guard set (`StartChargepointCharging`,
`src/internal/easee/controller.go:918`). This file was untracked in the main tree and committed in
the OpenSpec cleanup PR.
