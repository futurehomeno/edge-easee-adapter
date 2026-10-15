# PR #180 review fixes, round 3

## Why
- Files migrate.sh writes keep easee's primary group, still the legacy one when `usermod` fails, so that group could read the migrated secrets.

## What Changes
- postinst re-owns the data tree to `easee:futurehome` after the migration.

## Impact
- Affected specs: `packaging`
- Affected code: `package/debian/DEBIAN/postinst`
