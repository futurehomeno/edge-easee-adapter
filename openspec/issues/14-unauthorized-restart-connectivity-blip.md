# The unauthorized restart blips every charger's connectivity

Found in the review of PR #213.

Each unauthorized restart closes the shared connection, so `handleClientState(Disconnected)`
publishes a connectivity DOWN for every charger, followed by UP after the re-subscribe, and every
`checkConnection`-gated command or report fails in between. With a ~1 h token (issue 4) that is up to
one blip per hour on a site whose chargers need re-syncs (the 15 s re-sync after an adapter start or
stop, the stale-state watchdog).

Not suppressed: skipping the DOWN report for a deliberate restart would leave the chargers shown
connected if the reconnect then fails, because no further state change is emitted.

Open: is the blip visible or harmful in the app, or to energy guard (which waits up to 60 s for the
state change), and is a suppressed-with-timeout report worth the code?

2026-11-25: `charging-watchdog-window` (20-minute watchdog) removes most re-syncs during a session. What is left is
the re-sync 15 s after a start or stop: the first one after the connection's token has ended still restarts the
client. 3.3.1 beta logs of that day: 63 restarts on 7 of 98 sites.
