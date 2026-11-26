## 1. Upgrade
- [x] 1.1 Pin cliffhanger `v1.5.6-0.20261125115807-4a5e15820840`; build, vet and tests pass
- [ ] 1.2 Re-pin to the cliffhanger release before merge

## 2. Codes and logging
- [x] 2.1 Test: HTTP 401, 404, 400, 429, 500 and no answer give coded errors without URL, serial or body, and no log line
- [x] 2.2 Classify in `handleFailedResponse` and `transportError`; drop `logFailedResponse`
- [x] 2.3 Test and code: a lost stream is `unavailable`
- [x] 2.4 The unmapped phase mode error is returned only, not logged too

## 3. Late failure
- [x] 3.1 Test: a deferred offered current refused by Easee publishes `evt.error.report` with the command's `corid`
- [x] 3.2 Test: the binding survives the divert renaming the interface
- [x] 3.3 Bind/Unbind routes around the chargepoint routes; deferred commands keep their request; `router.NewErrorReport` on failure

## 4. Text
- [x] 4.1 Correct the "cliffhanger rejects under 6 A" comments and spec text (issue 56 for the divert left behind)
- [x] 4.2 Manifest declares `out evt.error.report string` (already, tested)

## 5. Review round (PR 245)
- [x] 5.1 Test and code: the HTTP failure cause carries the method and URL for the log
- [x] 5.2 Test and code: a deferred failure's report names no charger serial, coded or not
- [x] 5.3 Test and code: an unmapped phase mode is coded `unavailable` with the details in the cause
- [x] 5.4 Drop the Sub-Minimum Current Pauses delta, superseded by `offered-current-normal-path`; state that a report request still waits for the framework service's lock during a command's controller call
