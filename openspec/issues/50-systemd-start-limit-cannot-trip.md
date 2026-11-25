# 50: the systemd start limit of `easee.service` can never be reached

## What happened

Found by the `spec-coverage` change of 2026-11-25, read from the unit file, not seen on a hub.

`package/debian/usr/lib/systemd/system/easee.service` sets `StartLimitIntervalSec=10`, `StartLimitBurst=5`,
`Restart=always` and `RestartSec=10`. systemd gives up only after 5 starts inside 10 seconds, but with a 10 second
delay between restarts at most 2 starts fit in any 10 second window. The limit therefore never trips: a unit that
fails on every start is restarted every 10 seconds without end.

The same four lines are in other adapters (adax, device-monitor). The energy guard uses `StartLimitIntervalSec=120`,
where 5 starts at 10 second spacing do fit and the limit can stop a crash loop.

The `packaging` spec states the restart delay (requirement "Service Unit") but not the start limit, because it is not
clear whether "never give up" is the intent.

## Reproduce

Make the binary exit at once (for example a `data/config.json` that neither it nor its backup can load), start the
unit and watch `systemctl status easee`: it is restarted every 10 seconds and never reaches `start-limit-hit`.

## What would close it

A decision: if the adapter shall never give up, remove the two `StartLimit*` lines (or set `StartLimitIntervalSec=0`)
so the file says so; if a crash loop shall stop, raise `StartLimitIntervalSec` above `5 × RestartSec` (for example
120 as in the energy guard). Then add the limit to the "Service Unit" requirement.
