## 1. On-demand re-sync

- [x] 1.1 Failing test: a re-sync request for a subscribed charger invokes `SubscribeWithCurrentState` again.
- [x] 1.2 Failing test: a re-sync request for a charger that is not subscribed, or while the client is disconnected, invokes nothing.
- [x] 1.3 Implement the re-sync request in the SignalR manager.

## 2. Re-sync after start and stop

- [x] 2.1 Failing test: an accepted stop re-syncs the charger 15 s after the send.
- [x] 2.2 Failing test: an accepted start re-syncs the charger 15 s after the send.
- [x] 2.3 Failing test: a second command within the delay leaves one re-sync, 15 s after the later send.
- [x] 2.4 Failing test: a failed stop or start schedules no re-sync.
- [x] 2.5 Implement.

## 3. Charging watchdog

- [x] 3.1 Failing test: cached `charging` and no state or power observation for 5 minutes triggers a re-sync.
- [x] 3.2 Failing test: `DynamicChargerCurrent` and phase-current observations do not reset the watchdog.
- [x] 3.3 Failing test: a non-charging cached state never triggers the watchdog.
- [x] 3.4 Failing test: while the silence lasts, re-syncs are at least 5 minutes apart.
- [x] 3.5 Implement.

## 4. Power override

- [x] 4.1 Failing test: power above zero observed before a newer non-charging state reports that state.
- [x] 4.2 Test: power above zero observed at or after the state still reports `charging`.
- [x] 4.3 Implement.

## 5. Verify

- [x] 5.1 `go vet ./...`, `go build ./...`, `golangci-lint run`.
- [x] 5.2 `make test`.
- [x] 5.3 `openspec validate resync-state-after-charge-command --strict`.
- [ ] 5.4 Archive the change in the PR.
