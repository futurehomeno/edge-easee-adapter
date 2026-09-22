# grid-fault-and-logout

## Why

Two defects that both turn a transient failure into a durable wrong state, filed as
[#164](https://github.com/futurehomeno/edge-easee-adapter/issues/164) and
[#165](https://github.com/futurehomeno/edge-easee-adapter/issues/165).

- **A wiring fault at the wrong moment wipes the stored topology.** `#153` stopped the SignalR
  path from republishing a zero-phase fault, because writing it deletes `phases` and
  `sup_phase_modes` and leaves energy guard unable to phase-balance the charger. The HTTP factory
  path has no equivalent guard: `updateChargerConfigState` maps Easee grid types 50/51/52 through
  `ToFimpGridType` to `("", 0)`, `(TN, 0)` or `(IT, 0)`, and `thingFactory.Create` persists that.
  An adapter restart during a wiring fault therefore overwrites the last good topology on disk,
  and the next `Create` builds the chargepoint service with zero phases — which outlives the fault.
- **A failed logout still reports the user logged out.** `application.Logout` marks the lifecycle
  not-authenticated and not-configured even when `ClearCredentials` failed, so the tokens stay on
  disk. `Initialize` treats a non-empty credential store as a live session: it sets
  `AuthStateAuthenticated` and re-seeds the chargers, so the next hub restart silently logs the
  user back in.

## What Changes

- **The factory keeps the last known topology through a wiring fault.** A config fetch whose grid
  type yields zero phases leaves `GridType`, `Phases` and `PhaseMode` at their stored values, so
  nothing is persisted or advertised from a fault reading — the same rule the SignalR handler
  already applies, and the alarm remains the report for the fault.
- **Logout does not claim success while the credentials are still readable.** The lifecycle is
  marked logged-out only once the store is actually empty; if the clear failed but the store is
  empty anyway the logout stands, and if tokens remain the error is returned with the app left in
  the failed state rather than in a state a restart will silently reverse.

## Impact

- `src/internal/easee/controller.go` — no zero-phase write into the state
- `src/internal/app/app.go` — verify the store before marking logged out
