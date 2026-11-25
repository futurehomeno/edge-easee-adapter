# Design

## Chosen approach: one table-driven test, then the manifest edit

The test replaces `TestPackagedManifest_DeclaresRoutedLogAndNodeInterfaces` in
`src/cmd/manifest_internal_test.go` (package `cmd`) with
`TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces`. The old test checks five entries; the new
one checks the same five among the rest, so replacing it follows from the modified requirement.

    func TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces(t *testing.T) {
        t.Parallel()

        b, err := os.ReadFile("../../package/debian/usr/share/futurehome/easee/defaults/app-manifest.json")
        require.NoError(t, err)

        var m manifest.Manifest
        require.NoError(t, json.Unmarshal(b, &m))
        require.Len(t, m.Services, 1)

        declared := map[string]string{}
        for _, i := range m.Services[0].Interfaces {
            declared[i.MsgType] = i.Type + " " + string(i.ValueType)
        }

        require.Len(t, m.Services[0].Interfaces, len(declared), "a message is declared twice")
        assert.Equal(t, map[string]string{
            // bootstrap.DefaultRoute: config report and debug.Route
            cliffConfig.CmdConfigGetReport: "in null",
            cliffConfig.EvtConfigReport:    "out object",
            debug.CmdLogGetLevel:           "in null",
            // ... the 42 rows of the proposal's table, grouped per route bundle
        }, declared)
    }

- One map from `msg_t` to `"intf_t val_t"`; `assert.Equal` fails on an entry in only one of the two
  and on a wrong direction or value type, and its diff names every such entry. The length check
  catches a message declared twice, which the map would hide.
- Names come from cliffhanger's exported constants (`app.CmdAppGetState`, `cliffAdapter.CmdPingSend`,
  `debug.EvtLogFormatReport`, `config.CmdConfigGetReport`, ...), so a rename in cliffhanger breaks the
  build of the test, not silently the manifest.
- How the routes are enumerated: by hand, once. cliffhanger's `router.Routing` keeps its voters
  unexported and offers no listing, and `router.Router` matches messages only over MQTT, so a test
  cannot list the routes `routing.New` returns. The table is written from `routing.New` in its
  mounting order, one block per bundle: `bootstrap.DefaultRoute` (`config.RouteCmdConfigGetReport`,
  `debug.Route`), `app.RouteApp` (the always-on six plus `Login` and `Logout`, the capabilities
  `internal/app` implements), `cliffAdapter.RouteAdapter` (six commands). The per-setting
  `cliffConfig.RouteCmdConfig*` lines and `telemetry.Route` are left out by the requirement.
- The manifest edit then rewrites `services[0].interfaces` of the packaged manifest to the table,
  in the same block order, keeping the file's formatting.

About 75 lines of test (the table is most of it) and 115 more lines of JSON. No production code.

## Rejected alternatives
- Probe the real router over MQTT (the suite already starts a broker): send each declared command and
  wait for a reply. It needs the broker, runs handlers with side effects (`cmd.app.uninstall`), is
  slow, and can only confirm declared entries; it cannot find a routed message that is missing.
- Guard on `len(newRouting(cfg))`: counts device-service routes as well, fails with a bare number,
  and needs the service container set up. More lines for a weak signal.
- Build the list at runtime in `GetManifest` from the routes: the same enumeration problem, and it
  changes what the hub receives.
- Export a matcher from cliffhanger and derive the table: the real fix for drift, but a cross-repo
  change and a release; left as decision 5 in the proposal.
- List the per-setting config routes too: 63 more entries (about 315 lines of JSON) that nothing
  reads; decision 1.
