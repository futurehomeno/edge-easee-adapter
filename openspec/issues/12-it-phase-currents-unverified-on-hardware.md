# IT phase currents: not yet seen from a real charger

Found while fixing the phase currents per grid type (change `it-phase-currents`).

The T2 = L1 reading on IT rests on Easee's documentation (output phase names, backplate wiring).
Observation 182 was dropped by the receiver before that change, so no hub log shows a T2 value yet.

Doubts:
- To verify on an IT 3-phase charger: an `L1L2L3` session reports three loaded phases near the same
  current, an `L1L2` session loads `i1` and `i2`, `L3L1` loads `i1` and `i3`, and `i3` follows L3
  rather than staying at 0.
- On IT a single-phase session runs line to line, so after the change it loads two of `i1`..`i3`
  where it loaded one before. Energy guard's `carChargesOn1Phase` counts 1 or 2 loaded phases as a
  one-phase car, so that reading does not change; a real three-phase car now loads all three.
- A `TT` grid type in the props is mapped as IT. Easee detects TN or IT only, so `TT` can only come
  from stored state; a TT supply wired with a neutral would then be read from the wrong terminals.
