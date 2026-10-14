# Tasks

- [x] 1 Failing tests: manager asks the handler for a node report on subscribe result and disconnect; handler sends it on cloud/state online flips
- [x] 2 `Handler.ReportConnectivity()` queued on the report sender; called by the manager outside `m.mu`
- [x] 3 Tests green with -race, lint, `openspec validate connectivity-on-reconnect --strict`
