#!/bin/sh
#
# Migrate state from the legacy /opt/thingsplex paths to /var/lib/futurehome.
# The old tree is left in place so a downgrade to v2 still finds its state.
# Run as the easee user via fh-drop from postinst, which only invokes this
# script after probing (as root) that legacy state exists - so a failure to
# read or copy it below is an error, never a reason to skip.
# A failed migration aborts the install (set -e): better to block the upgrade
# than to start the service without the migrated state.
set -eu

if [ "$(id -u)" = 0 ]; then
	echo "easee: migrate.sh must run as easee, not root" >&2
	exit 1
fi

OLD_DATA=/opt/thingsplex/easee
OLD_LOGS=/var/log/thingsplex/easee
NEW_DATA=/var/lib/futurehome/easee
NEW_LOGS=/var/log/futurehome/easee

# data/ holds credentials (config.json, secrets.json): everything created below
# must be closed to others from the start. postinst's permission
# normalisation runs before this script, so modes set here are what stick.
umask 027

# Copy to a temp dir, sync, then rename into place: data/ is either absent or
# complete, an interrupted attempt is retried on the next run, and existing
# state is never deleted - once data/ exists the service owns it.
TMP="$NEW_DATA/.data.tmp"
rm -rf "$TMP"
if [ ! -e "$NEW_DATA/data" ] && [ -d "$OLD_DATA/data" ]; then
	echo "easee: migrating legacy state from $OLD_DATA"
	# cp -R (not -a): drop legacy owner bits; the umask above masks the
	# copied modes so nothing arrives world-readable.
	cp -R "$OLD_DATA/data" "$TMP"
	# Make the copy durable before the rename publishes it: the rename may
	# reach disk before the file data it points to.
	sync -f "$TMP"
	mv "$TMP" "$NEW_DATA/data"
fi

# The charging session history (buntdb) lives at the legacy work dir root, not
# under data/. Same temp+sync+rename pattern, and fatal on failure like the
# state above. No path rewrite needed: a session is {id,start,stop,energy}.
DB_TMP="$NEW_DATA/.data.db.tmp"
rm -f "$DB_TMP"
if [ ! -e "$NEW_DATA/data.db" ] && [ -f "$OLD_DATA/data.db" ] && [ ! -L "$OLD_DATA/data.db" ]; then
	echo "easee: migrating charging sessions from $OLD_DATA/data.db"
	cp "$OLD_DATA/data.db" "$DB_TMP"
	sync -f "$DB_TMP"
	mv "$DB_TMP" "$NEW_DATA/data.db"
fi

# Rewrite legacy absolute paths in a migrated config.json. Idempotent. The .bak is
# rewritten too: the service falls back to it when config.json is unreadable, and an
# untouched copy would point work_dir and log_file back at the legacy tree exactly
# when corruption recovery needs them right.
for CONF in "$NEW_DATA/data/config.json" "$NEW_DATA/data/config.json.bak"; do
	if [ -f "$CONF" ] && [ ! -L "$CONF" ]; then
		sed -i \
			-e "s|$OLD_DATA|$NEW_DATA|g" \
			-e "s|$OLD_LOGS|$NEW_LOGS|g" \
			"$CONF"
	fi
done

# Migrate the old log file. Best-effort and non-fatal: the log is history, not
# state, so failing to copy it must not block the upgrade. Copy when the
# target is missing or empty (-s): postinst pre-touches an empty log before
# this script runs, while a non-empty one was written by the service and must
# not be overwritten.
if [ -f "$OLD_LOGS/easee.log" ] && [ ! -L "$OLD_LOGS/easee.log" ] \
	&& [ ! -s "$NEW_LOGS/easee.log" ]; then
	cp "$OLD_LOGS/easee.log" "$NEW_LOGS/easee.log" \
		|| echo "easee: log migration incomplete (ignored)" >&2
fi

exit 0
