# 46: TestObservationsHandler_ModeThreeSetElsewhereAfterAReleaseIsLeftAlone is flaky

CI failed it on PR 240 (run 45409749754, commit d554c3b, a docs-only commit after a green run of the same code):
`Expected number of calls (1) of method SetPhaseMode does not match the actual number of calls (0)`.

Not caused by PR 240: `release_3.4` (3dbfe71) fails it 4 times in 200, the PR head once in 60.

Cause: the lock release after the disconnect is queued on the handler's sender goroutine (`handler.go`, "phase lock"),
and it skips itself when the cached phase mode changed while it waited. The test's next `PhaseMode` observations run
on the test goroutine at once, so the release sometimes finds the mode already changed and sends nothing.

Reproduce: `cd src && go test -race -count 200 ./internal/signalr/ -run 'TestObservationsHandler_ModeThreeSetElsewhereAfterAReleaseIsLeftAlone$'`.

Would close it: the test waits for the queued release (e.g. `flushReports` after the disconnect observation) before the
next observations, and the loop above passes.
