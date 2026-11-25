# 53 A slow session resumed without a mode can write a current below the minimum

Found in the review of PR 241. `StartChargepointCharging` (`src/internal/easee/controller.go`, ~905)
applies the `min_charging_current` floor only when `slowSince` is zero. The Start Charging
requirement (`openspec/specs/charging-control/spec.md`) exempts only "a start that is not slow".

Reproduce: start a session in mode `slow`; `cmd.charge.stop`; `cmd.current_session.set_current 3`
(kept under the user stop); `cmd.charge.start` without a mode. 3 A is written as the dynamic
current, below the 6 A minimum.

Close: decide whether such a resume keeps the sub-minimum value, then state it in the spec, or
apply the floor (or the pause) to it, test-first.
