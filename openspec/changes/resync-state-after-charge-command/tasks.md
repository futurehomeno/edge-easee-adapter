## 1. On-demand re-sync

- [ ] 1.1 Failing test: a re-sync request for a subscribed charger invokes `SubscribeWithCurrentState` again.
- [ ] 1.2 Failing test: a re-sync request for a charger that is not subscribed, or while the client is disconnected, invokes nothing.
- [ ] 1.3 Implement the re-sync request in the SignalR manager.

## 2. Re-sync after start and stop

- [ ] 2.1 Failing test: an accepted stop re-syncs the charger 15 s after the send.
- [ ] 2.2 Failing test: an accepted start re-syncs the charger 15 s after the send.
- [ ] 2.3 Failing test: a second command within the delay leaves one re-sync, 15 s after the later send.
- [ ] 2.4 Failing test: a failed stop or start schedules no re-sync.
- [ ] 2.5 Implement.

## 3. Charging watchdog

- [ ] 3.1 Failing test: cached `charging` and no power or phase-current observation for 5 minutes triggers a re-sync.
- [ ] 3.2 Failing test: `DynamicChargerCurrent` observations do not reset the watchdog.
- [ ] 3.3 Failing test: a non-charging cached state never triggers the watchdog.
- [ ] 3.4 Failing test: while the silence lasts, re-syncs are at least 5 minutes apart.
- [ ] 3.5 Implement.

## 4. Power override

- [ ] 4.1 Failing test: power above zero observed before a newer non-charging state reports that state.
- [ ] 4.2 Test: power above zero observed at or after the state still reports `charging`.
- [ ] 4.3 Implement.

## 5. Verify

- [ ] 5.1 `go vet ./...`, `go build ./...`, `golangci-lint run`.
- [ ] 5.2 `make test`.
- [ ] 5.3 `openspec validate resync-state-after-charge-command --strict`.
- [ ] 5.4 Archive the change in the PR.
