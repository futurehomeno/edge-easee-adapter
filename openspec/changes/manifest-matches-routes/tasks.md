# Tasks

## 1. Test first
- [ ] 1.1 In `src/cmd/manifest_internal_test.go`, replace `TestPackagedManifest_DeclaresRoutedLogAndNodeInterfaces` with `TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces`: the table of the proposal (41 messages, names from cliffhanger constants), compared with `services[0].interfaces` of the packaged manifest, plus the check that no message is declared twice
- [ ] 1.2 Run `cd src && go test ./cmd -run TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces`; it fails for the expected reason: the diff shows `cmd.app.get_full_state`, `cmd.app.full_state_report`, `cmd.auth.set_tokens`, `cmd.thing.inclusion` and `evt.auth.login_report` only in the manifest, `cmd.thing.delete` with value type `string`, and the 28 routed messages missing

## 2. Manifest
- [ ] 2.1 Rewrite `services[0].interfaces` of `package/debian/usr/share/futurehome/easee/defaults/app-manifest.json` to the table, in the same block order; leave the other blocks and the `testdata/` copies unchanged
- [ ] 2.2 Run the test again; it passes

## 3. Gate
- [ ] 3.1 In `src/`: `go vet ./...`, `go build ./...`, `golangci-lint run`
- [ ] 3.2 `openspec validate manifest-matches-routes --strict`
- [ ] 3.3 No version bump or changelog here: versions are bumped in the separate "Bump go modules and version" PRs

## 4. Close
- [ ] 4.1 After the merge: `openspec archive manifest-matches-routes`, and move issue 48 to `openspec/issues/archive/`
