# Spec corrections: statements that disagree with the code

## Why
The OpenSpec cleanup of 2026-11-25 compared the main specs with the code on `release_3.4`. One
requirement that no pending change touches states a figure the code does not use:

- `reporting` "Periodic Reporting Task" says the reporting task runs on `polling_interval` with a
  default of 10m. Since PR #221 (3.3.2) the getter falls back to 20m
  (`src/internal/config/config.go:255`) and the packaged `defaults/config.json` sets
  `"pollingInterval": "20m"`. The `configuration` spec already says 20m.

## What Changes
- `reporting` "Periodic Reporting Task": default 20m, as in the code. The code is the reference;
  no code and no test change.

## Not in this change
- The Configuration Routes requirement lacks `initial_charging_current` and `min_charging_current`.
  The pending change `configurable-minimum-current` already adds both, so it is fixed when that
  change is archived.

## Impact
- Affected specs: `reporting`
- Affected code: none
