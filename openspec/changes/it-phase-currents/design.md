# Design: phase currents per grid type

## Terminal meaning

| Terminal | TN | IT / TT |
| --- | --- | --- |
| T1 | PE | PE |
| T2 | N | L1 |
| T3 | L1 | L2 |
| T4 | L2 | L3 |
| T5 | L3 | not wired |

## Source
- https://developer.easee.com/docs/observation-ids: 182-185 are `INT CURRENT T2`..`T5`,
  "Calculated current RMS for input T2".."T5"; 190 `IN VOLT T1T2` and its siblings name the same
  terminals.
- https://developer.easee.com/docs/enumerations, Output Phase (110): `P1_T2_T3_TN` 1-phase (N+L1),
  `P1_T2_T4_TN` (N+L2), `P1_T2_T5_TN` (N+L3), `P1_T2_T3_IT` 1-phase (L1+L2), `P1_T2_T4_IT` (L1+L3),
  `P1_T3_T4_IT` (L2+L3), `P2_T2_T3_T4_IT` 2-phases on IT (L1+L2, L2+L3).
- Easee installer guides, backplate wiring (Charge Up:
  https://download.easee.com/m/1a77b7aa5bf8fc3c/original/EN_ChargeUp_InstallationGuide.pdf,
  Home/Charge: https://download.easee.com/m/475dfe81bbb5bbb4/original/NO_Home_Charge_IG.pdf):
  TN/TT 3-phase `PE N L1 L2 L3`, IT/TT 3-phase `PE L1 L2 L3 –`, TN 1-phase `PE N L1 – –`,
  IT/TT 1-phase `PE L1 L2 – –`.

## Decisions
- The grid type is read from the chargepoint props, like `handleOutPhase`: a topology republish
  that is still queued has replaced the props but not yet written the cache.
- While the props carry no grid type, the TN mapping applies. That is the mapping every charger
  used before this change, and the props carry a grid type from the stored state or the first
  grid type observation.
- TT is mapped as IT, as `OutputPhaseType.ToFimpState` does. Easee's detected grid types are TN or IT
  only: a TT supply wired with a neutral is detected as TN, one without as IT, so a TT grid type in
  the props can only come from stored state.
- A terminal outside the grid's lines returns no error: on TN, T2 is the neutral current Easee
  streams with the others, and an error is logged as a warning per observation.
- When the grid type changes, each phase keeps its cached value until its new terminal reports a
  reading newer than the cached one; the cache's timestamp guard does not know the terminal, so an
  older replayed reading is dropped. No reset: the thing is created with the grid type from the
  charger config, so a change while currents flow needs a failed config fetch with no stored state
  or a rewired charger, and the next reading while charging corrects it.
