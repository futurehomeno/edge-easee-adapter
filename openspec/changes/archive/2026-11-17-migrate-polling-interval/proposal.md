# Lift the 2.5 polling interval to 10m

## Why
- Up to 2.5.10 the packaged `pollingInterval` was `30s`; 2.6.1 shipped `10m`. The defaults are copied
  only on first install, so hubs first installed on 2.5 still poll every 30 s on 3.3.1.
- Each poll of an idle charger asks for its phase mode report, which has no output phase to answer
  from and fetches the charger config and site info over REST: four calls a minute per charger,
  seen on a 3.3.1 hub with two chargers on 2026-11-17.

## What Changes
- A config migration to version 9 rewrites a stored `pollingInterval` of exactly `30s` to `10m`,
  as the earlier superseded-default migrations do.

## Impact
- Affected specs: `configuration`
- Affected code: `src/internal/config/config.go`, `src/cmd/factory.go`
- A `30s` set on purpose over `cmd.config.set_polling_interval` is indistinguishable from the old
  default and is rewritten too.
