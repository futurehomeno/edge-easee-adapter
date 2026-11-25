# Tasks

## 1. Test first
- [x] 1.1 In `src/cmd/manifest_internal_test.go`, replace `TestPackagedManifest_DeclaresRoutedLogAndNodeInterfaces` with `TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces`: the table of the proposal (42 messages, names from cliffhanger constants), compared with `services[0].interfaces` of the packaged manifest, plus the check that no message is declared twice
- [x] 1.2 Run `cd src && go test ./cmd -run TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces`; it fails for the expected reason: the diff shows `cmd.app.get_full_state`, `cmd.app.full_state_report`, `cmd.auth.set_tokens`, `cmd.thing.inclusion` and `evt.auth.login_report` only in the manifest, `cmd.thing.delete` with value type `string`, and the 28 routed messages missing

## 2. Manifest
- [x] 2.1 Rewrite `services[0].interfaces` of `package/debian/usr/share/futurehome/easee/defaults/app-manifest.json` to the table, in the same block order; leave the other blocks and the `testdata/` copies unchanged
- [x] 2.2 Run the test again; it passes

## 3. Gate
- [x] 3.1 In `src/`: `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 3.2 `openspec validate manifest-matches-routes --strict`
- [x] 3.3 No version bump or changelog here: versions are bumped in the separate "Bump go modules and version" PRs

## 4. Close
- [x] 4.1 `openspec archive manifest-matches-routes` (in this PR, so the spec lands with the code), and move issue 48 to `openspec/issues/archive/`
