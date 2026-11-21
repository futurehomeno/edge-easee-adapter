# TN 3-phase reports two-phase legs it does not advertise

Found in the review of PR 210, older than it.

`outputPhaseModes` maps Easee output 20 `P2_T2_T3_T4_TN` to `NL1L2` and 21 `P2_T2_T4_T5_TN` to
`NL2L3`, but the TN 3-phase rows of `phaseModeMatrix` list only `NL1`, `NL2`, `NL3`, `NL1L2L3`.
A two-phase EV on a TN 3-phase charger makes `evt.phase_mode.report` carry `NL1L2` or `NL2L3`,
which `sup_phase_modes` does not contain, and `outputPhaseStale` drops that leg as soon as a newer
internal phase mode observation arrives while the charger is idle.

Doubts:
- Should the TN 3-phase auto row advertise `NL1L2` and `NL2L3`? They are not settable: Easee has no
  internal mode that asks for two phases, so a set would map to auto and the EV decides.
- Does cliffhanger or energy guard reject a reported mode outside `sup_phase_modes`? Not checked.
