# cache-manifest-chargers

## Why

During setup the app polls `cmd.app.get_manifest` several times a second and each call fetched
`GET /api/chargers`; Easee answered 429 and the device picker failed (3.1.4 hub log).

## What Changes

- The manifest device selector reuses a successful charger list for 10 s. Errors are not cached;
  login and logout drop the cached list. Login, Configure and re-seed keep fetching fresh.
