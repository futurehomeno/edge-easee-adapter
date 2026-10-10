# Read the output phase against the detected grid type

## Why
An Easee on an IT 1-phase grid (`detected_power_grid_type=5`) reports `output_phase=10`,
`P1_T2_T3_TN`, while it charges. The adapter maps that value on its suffix alone and reports `NL1`,
a neutral-referenced leg that does not exist on an IT grid, while `grid_type` says IT and
`sup_phase_modes` advertises only `L1L2`. The hub discards a report naming a mode outside
`sup_phase_modes`, and consumers such as Energy Guard read the wrong leg. The charger's own grid
detection is the reliable half: the TN/IT suffix names the same terminals either way.

## What Changes
- The output-phase observation is translated with the cached FIMP grid type. A TN-suffixed value
  on an IT or TT grid maps to its IT twin on the same terminals (10→11, 12→13, 20→22), and an
  IT-suffixed value on a TN grid maps to its TN twin.
- Values without a twin, and any value while the grid type is not yet known, keep today's mapping.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/model/model.go`, `src/internal/signalr/handler.go`
