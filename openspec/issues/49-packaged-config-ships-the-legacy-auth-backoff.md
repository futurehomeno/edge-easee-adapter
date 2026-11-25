# 49: the packaged config ships the legacy `authenticatorBackoff` object, not `auth_backoff`

## What happened

Found by the `spec-coverage` change of 2026-11-25, read from the files, not seen in a log.

`package/debian/usr/share/futurehome/easee/defaults/config.json` still carries the object v2.5.5 introduced:

    "authenticatorBackoff": { "initialBackoff": "1m", "repeatedBackoff": "5m", "finalBackoff": "10m",
                              "initialFailureCount": 4, "repeatedFailureCount": 8 }

v2.7 (#79) renamed the setting to `auth_backoff` / `auth_max_unauthorized` and added the migration 2→3
(`Config.MigrateAuthBackoff`, `src/internal/config/config.go`), but the packaged file was never changed. It also
carries no `config_version`, so every first start (and every start after an uninstall reset) runs the migrations
from version 0, and 2→3 converts the packaged legacy object into `auth_backoff`. That conversion is the only place
the failure counts 4 and 8 come from: without the object, `AuthenticatorBackoffStateful` reads a zero count as 1
(`backoffSettings.stateful`, `cmp.Or(…, 1)`), so the authenticator would back off after every failure instead of
after the fourth and the eighth.

`TestUninstall_resetAndReMigrate` (`src/cmd/reset_internal_test.go`) knows this ("the authenticator backoff is
rebuilt from the legacy block the defaults still ship"), but no spec says the packaged file must carry the legacy
shape, and the `configuration` requirement "Backoff Settings" names only 1m / 5m / 10m as the defaults.

## Reproduce

1. Fresh install: start the adapter with no `data/config.json`.
2. Read `data/config.json`: `auth_backoff` holds counts 4 and 8, and `authenticatorBackoff` is gone.
3. Remove `authenticatorBackoff` from the packaged file and repeat: `auth_backoff` is empty and the counts read as 1.

## What would close it

A decision whether the counts 4 / 8 are wanted. If yes: ship them under `auth_backoff` (and drop the legacy object),
or make 4 / 8 the getter fallback, and state them in "Backoff Settings". If no: drop the object and accept counts
of 1. Either way the packaged file then matches the current key names.
