# 41: TestEaseeAdapter failed once in `make test`

`make test` on the merge of `release_3.4` (with #237) into `feat/min-charging-current`, 2026-11-23: `TestEaseeAdapter`
(`src/cmd`) failed after 0.88 s, where a passing run takes about 11 s. The next `make test` and four runs of
`go test -race -count 1 ./cmd/` passed with the same tree. The failing log was not kept. Other sessions were running
tests of this adapter in other worktrees on the same machine at that time, so a shared resource (a port, a path
under the work directory) is the first thing to check.

Reproduce: not reproduced; run `make test` in two worktrees at once.

Closes when the cause is found, or when it has not come back in CI and locally by the next release.
