# postinst-futurehome-group

## Why

PR #180 review: on a hub upgraded from the thingsplex layout the `easee` user already exists, so
postinst skips `adduser --ingroup futurehome` and the user keeps its legacy primary group
`thingsplex`. The unit sets no `Group=` and migrate.sh runs as `easee`, so the migrated
`config.json.bak` and every `secrets.json` the adapter writes are `easee:thingsplex`, readable by
that group at the pinned 0640.

## What Changes

- postinst moves an existing `easee` user's primary group to `futurehome`, as mill, netatmo, sonos
  and tpflow already do.

## Impact

- Affected specs: `packaging` (new)
- Affected code: `package/debian/DEBIAN/postinst`
