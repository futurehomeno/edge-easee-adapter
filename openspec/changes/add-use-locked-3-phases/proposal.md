# Lock three phases during a session (`use_locked_3_phases`)

## Why
- A three-phase request maps to Easee internal mode 2 (auto), so the charger may still pick one
  phase. Installations that want a hard three-phase lock while a car charges have no way to get it.
- Most EVs refuse a 1-to-3 phase transition mid-session, so the lock must stay optional and must
  never be left on an idle charger.

## What Changes
- New boolean setting `use_locked_3_phases`, default `false`, with config get/set routes. With
  `false` nothing changes.
- With `true`, a three-phase request maps to internal mode 3 while a session is in progress
  (FIMP state `ready_to_charge`, `charging`, `suspended_by_ev`, `finished`, `error` or `requesting`:
  only `disconnected` ends it) and to auto otherwise.
- With `true`, a session start on auto re-sends the three-phase mode, which then locks; a session
  end or disconnect on mode 3 re-sends it, which then returns to auto. Single phase is never touched.

## Impact
- Affected specs: `phase-mode`, `configuration`
- Affected code: `src/internal/config/config.go`, `src/internal/routing/router.go`,
  `src/internal/model/mapper.go`, `src/internal/easee/controller.go`,
  `src/internal/signalr/handler.go`, packaged `defaults/config.json`
- Open points: issues 25, 26.
