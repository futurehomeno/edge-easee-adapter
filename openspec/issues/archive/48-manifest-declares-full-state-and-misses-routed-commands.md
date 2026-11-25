# 48: the app manifest declares full-state messages nobody routes and misses routed commands

Found in the OpenSpec cleanup of 2026-11-25. Read from the files, not seen in a log.

The `easee` service in `package/debian/usr/share/futurehome/easee/defaults/app-manifest.json`:

- declares `cmd.app.get_full_state` with `intf_t` `out`, although it is a command, and a
  `cmd.app.full_state_report`, which is named like a command but would be a report. Neither the
  adapter (`src/internal/routing/router.go`) nor cliffhanger v1.5.5 (`app/routing.go`) routes or
  sends either message.
- does not declare messages the adapter routes through cliffhanger: `cmd.app.get_state`,
  `cmd.config.get_extended_report`, `cmd.config.extended_set`, `cmd.app.uninstall`,
  `cmd.app.get_diag`, `cmd.auth.login`, `cmd.auth.logout` (from `app.RouteApp`; the app has
  `Login` and `Logout` in `src/internal/app/app.go`), `cmd.config.get_report` (from `bootstrap.DefaultRoute`),
  and the `cmd.config.get_*` / `cmd.config.set_*` routes for each setting in `router.go:44-79`,
  with their reports.
- declares `cmd.auth.set_tokens`, which cliffhanger routes only for an app with `Authorize`; this
  app has none, so the command is not handled.
- declares `evt.auth.login_report`, while cliffhanger's auth report is `evt.auth.status_report`.

The `app-manifest` spec only requires the log-level and single-node pairs, so it does not say
which of these must be listed.

Not known: whether any client (the Futurehome app, Thingsplex, playground tools) reads the
manifest's interface list to decide what it may send. If none does, the wrong entries are
harmless.

What would close it: decide which messages the manifest must list. Then remove or correct the
two full-state entries, add the routed ones, extend the `app-manifest` spec with them and the
manifest test in `src/cmd/manifest_internal_test.go`.

Resolved: 2026-11-25, OpenSpec change `manifest-matches-routes` (archived as `2026-11-25-manifest-matches-routes`): the manifest declares the 41 messages routed on the app service, and `TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces` compares the list with the routes both ways.
