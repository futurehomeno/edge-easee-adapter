# The app manifest declares exactly the messages the adapter routes

## Why
- The `easee` service in `package/debian/usr/share/futurehome/easee/defaults/app-manifest.json`
  lists messages that nothing routes or sends (`cmd.app.get_full_state` as `out`,
  `cmd.app.full_state_report`, `cmd.auth.set_tokens`, `cmd.thing.inclusion`, `evt.auth.login_report`)
  and leaves out most of what the adapter does route (issue 48).
- The `app-manifest` spec only requires five entries, and its test checks only those five, so nothing
  stops the list from drifting again.
- The list was copied from a template shared by the whole adapter fleet (see "What the research
  found"); it was never derived from the routes.

## What Changes
- `app-manifest` requirement "Routed Interfaces Declared" is widened from five entries to the whole
  list: exactly the messages routed on the app service, commands `in`, replies `out`, with value
  types; optional auth/reset commands only when the app implements them; per-setting
  `cmd.config.get_<setting>` / `set_<setting>` routes left out.
- The existing test in `src/cmd/manifest_internal_test.go` is replaced by one table-driven test that
  compares the manifest's interface list with the table of routed messages, both ways.
- The packaged manifest's `services[0].interfaces` is rewritten to that table: 41 entries instead of 18.
- No runtime code changes. No route is added or removed.

## The table (cliffhanger v1.5.5, `src/internal/routing/router.go`)

Routed by: D = `bootstrap.DefaultRoute` (config report + `debug.Route`), A = `app.RouteApp`
(the app has `Login` and `Logout`, no `Authorize`, no `Reset`), N = `cliffAdapter.RouteAdapter`.

| Interface (type, msg_t, val_t) | In manifest | Routed | Action |
|---|---|---|---|
| in `cmd.app.get_manifest` null | yes | A (also takes a string mode, `manifest_state` or `full`) | keep as null |
| out `evt.app.manifest_report` object | yes | A reply | keep |
| out `cmd.app.get_full_state` string | yes | no | remove |
| out `cmd.app.full_state_report` object | yes | no | remove |
| in `cmd.auth.set_tokens` str_map | yes | no (no `Authorize`) | remove |
| out `evt.auth.login_report` string | yes | no; the reply is `evt.auth.status_report` | fix name and type: out `evt.auth.status_report` object |
| in `cmd.network.get_all_nodes` null | yes | N | keep |
| out `evt.network.all_nodes_report` object | yes | N reply | keep |
| in `cmd.thing.get_inclusion_report` string | yes | N | keep |
| in `cmd.thing.inclusion` string | yes | no | remove |
| in `cmd.thing.delete` string | yes | N, reads a str_map `{"address"}` | fix value type: str_map |
| out `evt.thing.inclusion_report` object | yes | N reply | keep |
| out `evt.thing.exclusion_report` object | yes | N reply | keep |
| in `cmd.network.get_node` string | yes | N | keep |
| out `evt.network.node_report` object | yes | N reply | keep |
| in `cmd.log.get_level` null, in `cmd.log.set_level` string | yes | D | keep |
| out `evt.log.level_report` string | yes | D reply | keep |
| in `cmd.app.get_state` null / out `evt.app.state_report` object | no | A | add |
| in `cmd.config.get_extended_report` null / out `evt.config.extended_report` object | no | A | add |
| in `cmd.config.extended_set` object / out `evt.app.config_report` object | no | A | add |
| in `cmd.app.uninstall` null / out `evt.app.uninstall_report` object | no | A | add |
| in `cmd.app.get_diag` null / out `evt.app.diag_report` object | no | A | add |
| in `cmd.auth.login` object, in `cmd.auth.logout` null | no | A (reply `evt.auth.status_report`) | add |
| in `cmd.config.get_report` null / out `evt.config.report` object | no | D | add |
| in `cmd.network.reset` null / out `evt.network.reset_done` null | no | N | add |
| in `cmd.ping.send` string / out `evt.ping.report` object | no | N | add |
| in `cmd.log.get_format`, `get_file`, `get_revert_timeout` null; in `set_format`, `set_file`, `set_revert_timeout` string; out `evt.log.format_report`, `file_report`, `revert_timeout_report` string | no | D | add (decision 2) |
| `cmd.config.get_<setting>` / `set_<setting>` / `evt.config.<setting>_report` for the 18 settings of `router.go` and the 3 telemetry settings (63 messages) | no | yes | leave out (decision 1) |

Routes on device services (`chargepoint` from `thing.RouteCarCharger`, `routeOfferedPause`) are not
on the app service and stay out of this list.

## What the research found

Who reads `services[].interfaces` of an app manifest (decides how much the wrong entries matter):
- cliffhanger v1.5.5: nothing. `manifest.AppService.Interfaces` is only a field the loader fills and
  `cmd.app.get_manifest` sends back. No helper builds the list from routes or checks it. Its other
  `Interfaces` uses are device service specifications and `prime`.
- `~/proj/go`: tpflow, cloud-adapter, kind-owl-prime and zigbee-ad read `Interfaces` of device services
  (inclusion reports, vinculum), not of app manifests. ecollector and fhbutler only serve their own
  manifest. fimpee shows the number of interfaces and nothing else.
- `~/proj/mono-mobile` (the Futurehome mobile app, checkout of 2026-08-11) parses the list into
  `EdgeAppManifestService.interfaces`, but no code reads that field. Its check before sending a
  command (`req_fimp_action_factory.dart`) uses device services from the site structure. The app's
  commands to an edge app come from `ui_buttons[].req`, `auth` and `init_flow`.
- Not available locally: the Thingsplex UI, the cloud services. So no reader was found: the wrong
  entries are harmless as far as the local code shows, and removing entries carries little risk.
  The value of the change is a manifest that tells the truth to tools and future readers.

How the sibling adapters declare the list:
- mill, hoiax, netatmo, hue, tibber, sonos and adax carry the same template: `cmd.app.get_full_state`
  as `out` and `cmd.app.full_state_report`; netatmo, sonos and tibber also `evt.auth.login_report`;
  mill and tibber `cmd.log.level_report` (wrong name; hue fixed it). adax routes `cmd.system.sync`
  and `cmd.system.reconnect` itself, for its UI sync button. sensibo lists device interfaces on the
  app service.
- Tests: adax checks that each `intf_t` is `in` or `out`; hue checks one entry
  (`evt.log.level_report`); this repo checks five. None compares the manifest with the routes, and
  neither the fleet nor cliffhanger has a helper for it. Reused: cliffhanger's `manifest.Manifest`
  type to parse the file (as adax and hue do) and cliffhanger's exported message constants for the
  names.

## Decisions (user, 2026-11-25)
The user approved the recommendations:
1. Per-setting config routes (`cmd.config.get_<setting>` / `set_<setting>` and their
   `evt.config.<setting>_report`, 63 messages here, the telemetry settings included) stay out of the
   interface list. The `configuration` spec lists them; the manifest's `configs` block plus
   `cmd.config.extended_set` is what the UI uses.
2. All of `debug.Route` is listed (level, format, file, revert timeout: 12 messages), so the rule stays
   "everything routed on the app service except per-setting config".
3. Declared but unrouted commands (`cmd.auth.set_tokens`, `cmd.thing.inclusion`, the full-state pair)
   are removed; no route is added for them.
4. The manifest copies under `testdata/` stay as they are; the test reads the packaged file only.
5. No automatic route discovery here: `router.Routing` keeps its voters unexported. A later
   cliffhanger change could export a matcher (for example `Routing.Handles(*fimpgo.Message) bool`),
   and this test would then derive its table.
6. The same template errors in the seven sibling adapters: one change per repo after this one lands,
   with the same requirement text.

## Impact
- Affected specs: `app-manifest` (requirement "Routed Interfaces Declared" modified).
- Affected files: `src/cmd/manifest_internal_test.go` (rewritten, 46 → 79 lines),
  `package/debian/usr/share/futurehome/easee/defaults/app-manifest.json` (interfaces 104 → 246 lines).
- No behaviour change on the hub: routes, replies and the manifest's other blocks stay the same.
- Closes issue 48.
