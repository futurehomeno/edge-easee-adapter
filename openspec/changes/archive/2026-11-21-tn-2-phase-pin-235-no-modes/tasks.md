# Tasks

## 1. One place decides
- [x] 1.1 Write a failing test: `SettablePhaseModesFor` gives none for grid type 30, the nominal TN
  2-phase row for 2, 36, 38 and for an unknown raw type on TN with 2 phases
- [x] 1.2 Add `model.SettablePhaseModesFor`, run the test green

## 2. Thing creation
- [x] 2.1 Write a failing test: a charger created on grid type 30 advertises no `sup_phase_modes`
  and persists raw type 30; 2, 36, 38 advertise the row; a stored state without the raw type keeps
  the row, one with raw type 30 advertises none
- [x] 2.2 Carry the raw type in `State`, set it in `updateChargerConfigState`, build
  `sup_phase_modes` with `SettablePhaseModesFor`, run the test green

## 3. Grid type observation
- [x] 3.1 Write a failing test: 2 to 30 and 30 to 2 republish; 30 on 30, 2 on 2, 36 and 38 on 2
  do not; 30 on a state without the raw type republishes; a new topology on 30 drops the modes; a
  dropped republish of 2 to 30 is retried
- [x] 3.2 Seed the last advertised raw type from `PhaseStore`, compare its settable modes, commit it
  in `onPublished`, publish `SettablePhaseModesFor`, run the test green
- [x] 3.3 Confirm in cliffhanger that a chargepoint without `sup_phase_modes` rejects
  `cmd.phase_mode.set` and sends no `evt.phase_mode.report`

## 4. Gate
- [x] 4.1 `go test ./internal/model/ ./internal/signalr/ ./internal/easee/`
- [x] 4.2 `go build ./...`, `go vet ./...`, `golangci-lint run ./...`
- [x] 4.3 `openspec validate tn-2-phase-pin-235-no-modes --strict`

## 5. Existing test
- [x] 5.1 `TestController_UpdateState` "a healthy reading still replaces the stored topology"
  compares the whole refreshed state, which now carries the raw type: its expected state gains
  `DetectedGridType: GridTypeTN3Phase`
