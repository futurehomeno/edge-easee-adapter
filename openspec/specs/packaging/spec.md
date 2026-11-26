# packaging Specification

## Purpose
How the Debian package installs the adapter, runs it under systemd, migrates the state of the legacy
thingsplex layout and sets up the service user and its files.
## Requirements
### Requirement: Service User Group
The package SHALL give the `easee` user the primary group `futurehome`, also when the user already
exists from the thingsplex layout, so that the files the adapter and the migration write carry the
`futurehome` group. After the state migration postinst SHALL re-own the data tree to
`easee:futurehome`.

#### Scenario: upgraded from the thingsplex layout
- **WHEN** postinst configures a hub where the `easee` user already exists
- **THEN** it sets the user's primary group to `futurehome`

#### Scenario: group change fails
- **WHEN** postinst cannot change the existing user's primary group
- **THEN** it warns on stderr and the install continues

#### Scenario: migrated files
- **WHEN** postinst has run the state migration
- **THEN** it re-owns the data tree to `easee:futurehome`, whatever group the migration wrote with

#### Scenario: re-own fails
- **WHEN** postinst cannot re-own the migrated data tree
- **THEN** it warns on stderr and the install continues

#### Scenario: fresh install
- **WHEN** postinst configures a hub without the `easee` user
- **THEN** it creates the user with primary group `futurehome`

### Requirement: Package Contents
The Debian package `easee_<version>_<arch>.deb` (arch `armhf` or `amd64`) SHALL install the binary at
`/usr/bin/easee`, the migration helper at `/usr/lib/futurehome/easee/migrate.sh`, the read-only defaults
(`config.json`, `adapter.json`, `app-manifest.json`) under `/usr/share/futurehome/easee/defaults/`, the unit
`/usr/lib/systemd/system/easee.service`, the root-owned parents `/var/lib/futurehome` and `/var/log/futurehome`,
and the symlink `/var/lib/futurehome/easee/defaults` pointing at `/usr/share/futurehome/easee/defaults`. The
symlink SHALL be package content, so dpkg removes it on remove and restores it on reinstall. The package SHALL
depend on `fh-drop`. The build SHALL stage only files tracked by git under `package/debian`, so a stray file left
there never ships.

#### Scenario: package built
- **WHEN** `make deb-arm` or `make deb-amd` builds the package
- **THEN** it holds the binary, the helper, the defaults, the unit and the defaults symlink, and its control file
  depends on `fh-drop`

#### Scenario: untracked file in the package tree
- **WHEN** a file under `package/debian` is not tracked by git
- **THEN** it is not in the built package

### Requirement: Service Unit
The systemd unit SHALL run `/usr/bin/easee -c /var/lib/futurehome/easee` as user `easee` and group `futurehome`,
with `/var/lib/futurehome/easee` as the working directory, SHALL restart it 10 seconds after any exit
(`Restart=always`, `RestartSec=10`), SHALL send its standard output and error to the journal, because the file log
opens only after the configuration has loaded, and SHALL be wanted by `multi-user.target`.

#### Scenario: the adapter exits
- **WHEN** the adapter process exits, for example after a fatal configuration error or an MQTT error it cannot
  recover from
- **THEN** systemd starts it again 10 seconds later

#### Scenario: error before the log file opens
- **WHEN** the adapter prints an error before its configuration is loaded
- **THEN** the error is in the journal of `easee.service`

### Requirement: Service Start And Stop In Maintainer Scripts
`preinst` SHALL stop a running `easee.service` before the package contents are replaced, on install and on
upgrade, but not on `abort-upgrade`, because postinst does not start the service again after a rolled-back upgrade.
`postinst configure` SHALL enable the unit, reload systemd and start the service, ignoring a failure of any of the
three. `prerm remove` SHALL stop and disable the unit; `prerm` on upgrade SHALL leave it running. `postrm` SHALL
reload systemd. `preinst`, `prerm` and `postrm` SHALL touch systemd only where systemd runs (`/run/systemd/system`
exists). `postinst` SHALL do nothing for `abort-upgrade`, `abort-remove` and `abort-deconfigure`, and SHALL fail on
any other argument.

#### Scenario: upgrade
- **WHEN** the package is upgraded while the service runs
- **THEN** preinst stops it, and postinst enables and starts the new version

#### Scenario: rolled-back upgrade
- **WHEN** dpkg calls preinst with `abort-upgrade`
- **THEN** the running service is not stopped

#### Scenario: remove
- **WHEN** the package is removed
- **THEN** prerm stops and disables the unit and postrm reloads systemd

#### Scenario: unknown postinst argument
- **WHEN** postinst is called with an argument other than `configure` or an `abort-*` one
- **THEN** it prints an error and exits with status 1

### Requirement: Purge Removes All State
`postrm purge` SHALL remove `/var/lib/futurehome/easee`, `/var/log/futurehome/easee` and the legacy trees
`/opt/thingsplex/easee` and `/var/log/thingsplex/easee`, because the migration leaves the legacy tree in place for a
downgrade, so purge is the only step that removes the credentials and sessions it still holds. A plain remove SHALL
keep all four.

#### Scenario: purge
- **WHEN** the package is purged
- **THEN** the data, the log and both legacy trees are gone

#### Scenario: remove without purge
- **WHEN** the package is removed but not purged
- **THEN** the data, the log and the legacy trees stay

### Requirement: Directory Layout At Configure
`postinst configure` SHALL create `/var/lib/futurehome/easee` and `/var/log/futurehome/easee` as `easee:futurehome`
with mode 0755, pre-create the log file `easee.log`, re-own both trees to `easee:futurehome` and set the log tree to
`u=rwX,go=rX` before the state migration runs, so that root-owned leftovers of an interrupted install cannot stop
the unprivileged migration. It SHALL create `data/` only after the migration, with mode 0755, because the migration
runs only while `data/` is absent. After the migration it SHALL set every directory under the data tree to 0755 and
remove group and other write from every file, and SHALL NOT widen any file further, so the secrets keep the mode
they were written with.

#### Scenario: fresh install
- **WHEN** postinst configures a hub with no data tree
- **THEN** the data root, `data/` and the log directory exist, owned by `easee:futurehome`, mode 0755

#### Scenario: modes after migration
- **WHEN** the migration wrote `data/secrets.json` with mode 0640
- **THEN** after postinst its directories are 0755 and the file is still 0640

### Requirement: Symlink Defences In postinst
Because the data root and the log directory are writable by `easee`, postinst, running as root, SHALL NOT follow
a symlink planted there:
- the defaults symlink SHALL be re-created on every configure when it does not point at
  `/usr/share/futurehome/easee/defaults`, whatever sits at its path;
- a symlink at `data/` SHALL be removed with a warning and replaced by a real directory; when `data/` is still not a
  real directory afterwards, postinst SHALL fail. The removal runs after the migration, so a symlink at `data/` that
  resolves to a directory makes the migration see `data/` as present and skip;
- a symlink at `easee.log` SHALL be removed with a warning, and the log file SHALL be created in the root-owned
  parent and hard-linked into place, which fails rather than follows when anything already sits at the path; postinst
  ignores that failure, so a directory or other non-symlink left at `easee.log` stays and no log file is created;
- a symlink at `data/secrets.json`, `data/secrets.json.bak` or `data/config.json.bak` SHALL be removed with a
  warning, and postinst SHALL NOT change the mode of those files by path; the adapter sets their mode when it starts
  (configuration "Secret File Modes Pinned At Startup").

The mode pass over the data tree (`find -type d -exec chmod`) checks the type and changes the mode in two separate
path lookups, so a process running as `easee` during configure can still swap a checked directory for a symlink in
between; closing that window is issue 54.

#### Scenario: defaults replaced by a directory
- **WHEN** `/var/lib/futurehome/easee/defaults` is a real directory or points elsewhere
- **THEN** postinst replaces it with the symlink to `/usr/share/futurehome/easee/defaults`

#### Scenario: planted data symlink
- **WHEN** `data/` is a symlink at configure
- **THEN** postinst warns, removes the link and creates a real `data/` directory, leaving the link target untouched

#### Scenario: planted log symlink
- **WHEN** `easee.log` is a symlink at configure
- **THEN** postinst warns and removes it, and the file it pointed at is neither created, re-owned nor re-moded

#### Scenario: planted secrets symlink
- **WHEN** `data/secrets.json` is a symlink after the migration
- **THEN** postinst warns and removes it

### Requirement: Legacy State Migration Trigger
`postinst configure` SHALL run `migrate.sh` as user `easee` through `fh-drop` when, probed as root, the legacy
directory `/opt/thingsplex/easee/data` or the legacy session file `/opt/thingsplex/easee/data.db` exists. Probing as
root SHALL keep an untraversable legacy parent from silently skipping the trigger. A failed migration SHALL fail
the configure, so the service does not start without the migrated state. `migrate.sh` itself probes as `easee`, so
legacy state `easee` cannot reach is still skipped with success; making that fail is issue 55.

#### Scenario: upgrade from the thingsplex layout
- **WHEN** postinst configures a hub with `/opt/thingsplex/easee/data`
- **THEN** it runs `migrate.sh` as `easee`

#### Scenario: only the legacy sessions are left
- **WHEN** `/opt/thingsplex/easee/data` is gone but `/opt/thingsplex/easee/data.db` exists
- **THEN** it runs `migrate.sh`

#### Scenario: no legacy state
- **WHEN** neither legacy path exists
- **THEN** `migrate.sh` is not run

### Requirement: Legacy State Migration
`migrate.sh` SHALL copy `/opt/thingsplex/easee/data` to `/var/lib/futurehome/easee/data` and
`/opt/thingsplex/easee/data.db` to `/var/lib/futurehome/easee/data.db`, each only when its target does not exist,
through a temporary copy that is synced and then renamed into place, so the target is either absent or complete
and a broken attempt is retried on the next run. It SHALL never overwrite or delete state already in the new tree,
and SHALL leave the legacy tree in place for a downgrade. In the migrated `data/config.json` and
`data/config.json.bak` it SHALL rewrite the legacy data and log paths to the new ones; a re-run SHALL change nothing
more. It SHALL copy the legacy log `/var/log/thingsplex/easee/easee.log` when the new log is missing or empty, and a
failed log copy SHALL NOT fail the migration. A failed copy of `data/` or `data.db` SHALL fail it.

#### Scenario: upgrade from the thingsplex layout
- **WHEN** the legacy tree holds `data/config.json`, `data/config.json.bak`, `data/adapter.json`, `data.db` and a log
- **THEN** all are in the new tree, both config files point at the new paths, and the legacy files are unchanged

#### Scenario: state already migrated
- **WHEN** `data/`, `data.db` and a non-empty log already exist in the new tree
- **THEN** they are kept, only the legacy paths in the config files are rewritten

#### Scenario: legacy data directory removed
- **WHEN** only the legacy `data.db` exists
- **THEN** it is copied and no `data/` is created

### Requirement: Legacy State Protection
`migrate.sh` SHALL refuse to run as root. Everything it creates SHALL be closed to others (umask 027), and it SHALL
remove all access for others from the legacy `data/` tree, which older packages left world-readable while its
`config.json` still carried the tokens, ignoring a failure to do so; the files SHALL stay readable by `easee` for a
downgrade. While the new `data/` does
not exist, a symlink at the legacy `data/` SHALL fail the migration, because postinst found legacy state through it and skipping would leave the
hub logged out for good. A symlink at the legacy `data.db` or the legacy log SHALL be skipped, and a migrated config
file that is a symlink SHALL NOT be rewritten.

#### Scenario: run as root
- **WHEN** `migrate.sh` runs with user ID 0
- **THEN** it prints an error and exits with status 1

#### Scenario: modes of the copy and of the legacy tree
- **WHEN** the legacy `data/` is 0755 with a `config.json` of 0644
- **THEN** no file or directory of the new tree and none of the legacy `data/` tree grants others any access

#### Scenario: symlinked legacy data directory
- **WHEN** the legacy `data/` is a symlink, also one into a directory `easee` cannot enter
- **THEN** the migration fails with "is a symlink" and no `data/` is created in the new tree

#### Scenario: symlinked legacy sessions and log
- **WHEN** the legacy `data.db` and the legacy log are symlinks
- **THEN** neither is copied and the rest of the state is migrated

### Requirement: Packaged Default Configuration
The packaged `defaults/config.json` SHALL carry no `config_version`, so a first start, and the first start after an
uninstall reset, runs every configuration migration. It SHALL ship the MQTT broker `tcp://localhost:1883` with the
client ID prefix `easee` and no username or password, `work_dir` `/var/lib/futurehome/easee`, `log_file`
`/var/log/futurehome/easee/easee.log`, `log_level` `info`, `log_format` `budzik`, the Easee API at
`https://api.easee.com`, the SignalR service at `https://streams.easee.com`, `slowChargingCurrentInAmperes` 10, empty
tokens and no telemetry block.

#### Scenario: first start
- **WHEN** the adapter starts with no `data/config.json`
- **THEN** it loads the packaged defaults, reaches the latest config version and talks to `https://api.easee.com`

#### Scenario: uninstall reset
- **WHEN** the configuration is reset on uninstall and the adapter starts again
- **THEN** the packaged values are back and every migration runs again

