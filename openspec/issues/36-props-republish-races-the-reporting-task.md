# 36: the chargepoint props republish races cliffhanger's reporting task

Found while running `make test` for issue 33 (not caused by it): `go test -race ./cmd/` failed once in five runs.

`republishChargepointProps` (src/internal/signalr/handler.go, `service.Specification().Props = ...`) writes the props
on the observation drain while cliffhanger's reporting task reads them in `chargepoint.(*service).SupportedStates`
(tasks.go:41) with no lock between them.

Reproduce: `cd src && for i in 1 2 3 4 5; do go test -race -count=1 ./cmd/; done`; one run reports `DATA RACE` with
those two stacks.

Closes when the props are replaced through a path cliffhanger synchronises (or the reporting task cannot run during
the republish), and the loop above passes.

Seen again in CI on PR 240 (run 45398162517, 2026-11-24, commit 6a2adce), in `TestEaseeAdapter/Inclusion report
updated - different grid type`: the read side was `chargepoint.(*service).SendMaxCurrentReport` on the handler's sender
goroutine. The run on the next commit passed.
Seen again on PR 240 run 45419346856 (commit 9c1a7c8), the same two stacks; the rerun passed.
