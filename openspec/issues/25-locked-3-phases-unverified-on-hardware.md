# use_locked_3_phases: not yet seen on a real charger

Found while adding `use_locked_3_phases` (change `add-use-locked-3-phases`).

Only unit tests cover it. Unverified:
- Mode 3 written right at session start (`awaiting_start`, before charging): does the charger accept
  it, does the car start charging on three phases, and does a car that cannot take three phases
  stall instead of falling back?
- A session that goes straight to `charging` gets mode 3 while charging, so the controller bounces it
  (stop, mode write, resume): one extra ~30 s pause at every such start.
- Auto written at disconnect: accepted while no car is plugged in, and the echo arrives?
- A car that charges on one phase while the lock is on (single-phase car): Easee's behaviour on mode
  3 is not documented here.

How to verify: enable the setting on a test hub with a three-phase and a single-phase car, plug in,
charge, unplug; check the charger config's phase mode and the adapter log at each step.

Closes when: those steps are seen on hardware and the outcome is recorded here.
