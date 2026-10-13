# Log the phase mode by name

## Why
`Set phase mode to 2` is the line the adapter writes when it posts a phase-mode change to Easee.
The `2` is Easee's *internal* phase mode, an implementation detail of their API. Reading a hub log
it says nothing: 2 is auto, which on a 3-phase TN grid is how a three-phase request is expressed,
so the most common phase change in the field logs as an opaque digit. Diagnosing a phase-mode
complaint currently means knowing the 1/2/3 encoding and the direction of the mapping by heart.

## What Changes
- `SetPhaseMode` logs the internal mode with its name alongside the number, e.g.
  `Set phase mode to auto (2)`, so the line is readable without the mapping table and still
  carries the value that was actually posted.
- The phase-mode observation logs `Phase mode=auto (2)` instead of `Connected phases=2`, which
  named the wrong quantity: the value is Easee's internal mode, not a count of connected phases,
  so the old line read as a contradiction whenever a single-phase charger reported 2.
- A name is defined for each of Easee's three internal modes; an unrecognised value keeps the
  number and is named `unknown`, so a future Easee mode can never make the line lie.

## Impact
- Affected specs: `phase-mode`
- Affected code: `src/internal/model/mapper.go`, `src/internal/api/client.go`,
  `src/internal/signalr/handler.go`
- Log-only. No behaviour, API call or cached value changes.
