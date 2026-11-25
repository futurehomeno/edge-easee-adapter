# 51: some settings set over FIMP take effect only after an adapter restart

## What happened

Found by the `spec-coverage` change of 2026-11-25, read from the code, not seen on a hub.

`cmd.config.set_<key>` stores the value and answers with the new value (`configuration` "Configuration Routes"), but
several values are read only once, when the services are built at start:

| Setting | Read once in |
|---|---|
| `polling_interval` | `tasks.New` (`src/internal/tasks/tasks.go`): the reporting task interval |
| `http_timeout` | `getHTTPClient` (`src/cmd/factory.go`): the `http.Client` timeout |
| `easee_base_url` | `getEaseeHTTPClient` (`src/cmd/factory.go`): the base URL of every REST call |
| `signalr_initial_backoff`, `signalr_repeated_backoff`, `signalr_final_backoff`, `signalr_initial_failure_count`, `signalr_repeated_failure_count` | `signalr.NewClient` (`src/internal/signalr/client.go`): the reconnect backoff; the per-charger subscribe retry in `manager.go` does read them again |

So a set is reported as done while the running adapter keeps the old value until the next restart. The other routed
settings (`current_wait_duration`, `slow_charging_current_in_amperes`, `initial_charging_current`,
`min_charging_current`, `use_locked_3_phases`, `signalr_base_url`, `signalr_conn_creation_timeout`,
`signalr_keep_alive_interval`, `signalr_timeout_interval`, `signalr_invoke_timeout`) are read on each use.

## Reproduce

Send `cmd.config.set_polling_interval` with `1m` and watch the reporting task: it keeps the old interval until the
adapter is restarted.

## What would close it

A decision: either the listed settings are documented as "applied at the next start" (and the spec says so), or
they are read on each use / the affected services are rebuilt on a change.
