# PR 209 (advertise every leg): two open questions

## 1. Energy guard version shipped next to easee 3.3

Advertising every leg relies on energy guard learning the charger's leg from a mismatching
`evt.phase_mode.report`. That is `recordFixedPhaseMode`, core-energy-guard #309 (2026-09-06), first
tagged in v4.4.3.

With an older energy guard and a TN charger wired to L3: it picks `NL1` from the widened list, the
adapter answers `NL3`, and it asks again every balance tick. The adapter no-ops each request (the Easee
mode is already single phase), so the cost is command and log noise, not session bounces. This is the
loop #134 removed by narrowing the list.

Closed 2026-11-17: the oldest deployed energy guard is v4.4.6, so every hub has
`recordFixedPhaseMode`.

## 2. PR 209 carries #207 into release_3.3

The branch was cut from `release_3.2` and merged `main`; `release_3.3` was cut before #207. So the PR
diff also holds the #207 charging-control spec change and its archive directory
(`openspec/changes/archive/2026-10-22-pr180-start-echo-spec`). The text matches the release_3.3 code
and is already on `main`, so it has to reach release_3.3 anyway.

Open: leave it in this PR (PR body should then say so), or rebase the branch onto `release_3.3`
(force-push) and bring #207 over separately?
