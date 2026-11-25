# 55 migrate.sh exits 0 when easee cannot reach the legacy state postinst found

postinst probes `/opt/thingsplex/easee/data` and `data.db` as root and then runs `migrate.sh` as `easee`.
When an ancestor is untraversable for `easee`, the script's `[ -L ]`, `[ -d ]` and `[ -f ]` are false,
it migrates nothing and exits 0; `create_data_dir` then creates an empty `data/`, which disables the
migration for good and starts the adapter logged out (PR #242 review, P2).

Reproduce: legacy tree with `/opt/thingsplex/easee` at mode 0700 owned by root; run postinst configure.

Close: postinst tells `migrate.sh` what it found (e.g. an env flag), and the script fails when it cannot
see it; test in `src/cmd/` with the script against a temp tree. Then drop the limitation from packaging
"Legacy State Migration Trigger".
