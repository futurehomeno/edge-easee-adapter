# 36: the chargepoint props republish races cliffhanger's reporting task

Found while running `make test` for issue 33 (not caused by it): `go test -race ./cmd/` failed once in five runs.

`republishChargepointProps` (src/internal/signalr/handler.go, `service.Specification().Props = ...`) writes the props
on the observation drain while cliffhanger's reporting task reads them in `chargepoint.(*service).SupportedStates`
(tasks.go:41) with no lock between them.

Reproduce: `cd src && for i in 1 2 3 4 5; do go test -race -count=1 ./cmd/; done`; one run reports `DATA RACE` with
those two stacks.

Closes when the props are replaced through a path cliffhanger synchronises (or the reporting task cannot run during
the republish), and the loop above passes.
