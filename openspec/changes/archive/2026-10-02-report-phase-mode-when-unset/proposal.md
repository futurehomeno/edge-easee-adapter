# Report a phase mode when the charger's internal mode is unset

## Why
A beta hub (charger UKHWLRNR, v3.1.4, TN 1-phase) logged `unable to map phase modes ...
internal_phase_mode=0 phases=1` 38 times, and every energy guard `cmd.phase_mode.get_report` and
periodic report failed with it, from each restart until the first charging session cached an
output phase. Easee's charger config returns `phaseMode` 0 (null) for this model.

The inclusion report advertises `sup_phase_modes` from the auto row (`NL1` here), but the report
fallback maps the internal mode through the matrix, whose zero-mode guard yields nothing. The
persisted-phase fallback sits behind that empty result and is never reached. v2.8.0 advertised
nothing for this charger, so the failing report is a regression of the advertised list.

## What Changes
- When the refetched charger state carries internal mode 0, the report falls back to the
  advertised modes instead of the matrix row: the persisted phase when it is advertised, otherwise
  the first advertised mode. A TN 1-phase charger reports `NL1`, matching its `sup_phase_modes`.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/easee/controller.go`
