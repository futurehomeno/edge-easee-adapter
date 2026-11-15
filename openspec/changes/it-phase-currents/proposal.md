# Phase currents follow the grid type's line terminals

## Why
- Easee reports one current per input terminal, observations 182-185 `INT CURRENT T2`..`T5`. The
  handler maps T3, T4 and T5 to `i1`, `i2` and `i3` on every grid and filters 182 out.
- That is right on TN only. The backplate is PE, N, L1, L2, L3 on TN and PE, L1, L2, L3 on IT/TT,
  so on IT the lines are T2, T3 and T4: `L1` is never read, `L2` and `L3` are reported as `i1` and
  `i2`, and `i3` (the unwired T5) stays at 0.
- Seen on an IT 3-phase hub (energy guard's view, `i1/i2/i3`): `L1L2` gave 20.26/0.01/0.01, `L3L1`
  gave 0.01/23.83/0.01, and `L1L2L3` gave 17.12/17.25/0.01 at 6.5 kW, which is three lines at 17 A
  (√3 × 230 V × 17 A = 6.8 kW), not two. Energy guard counts loaded phases, so it took a
  three-phase session on IT for a car that does not charge on three phases.

## What Changes
- Observation 182 (in-current T2) is supported.
- The terminal-to-phase mapping follows the grid type in the chargepoint props: T2, T3, T4 on IT
  and TT; T3, T4, T5 on TN and while the grid type is unknown.
- A terminal that carries no line on that grid (T2 on TN, T5 on IT/TT) is ignored without an error.

## Impact
- Affected specs: `reporting`, `observation-streaming`
- Affected code: `src/internal/model/model.go`, `src/internal/signalr/handler.go`
