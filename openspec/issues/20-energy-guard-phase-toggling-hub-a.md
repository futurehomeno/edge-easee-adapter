# Energy guard toggles NL3 <-> NL1L2L3 every 20-40 min (hub with EHAZ4B5Y, EHTSKCHA)

Seen on v3.1.4 (logs 2026-10-16..2026-11-14): 256 `cmd.phase_mode.set` in four weeks, up to 32 a day.
Each switch pauses the session ~30 s (stop, mode write, resume). The adapter applies every request
and the charger reports the requested mode, so the adapter is not at fault; 3.2.5 does not change
this path apart from re-syncing state 15 s after the pause/resume.

Open: is this toggling the "phase changing" problem the user reported? If so, it is energy guard's
switching hysteresis, not the Easee adapter.
