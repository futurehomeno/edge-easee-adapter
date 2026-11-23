# 35: the end of a user stop rests on assumptions not checked on a charger

Found while making a user stop hold until `cmd.charge.start` (issue 33, `configurable-minimum-current`).

- Clocks: the stop is stamped with the hub clock, the charger states that end it with Easee's observation time. A
  skew of a few seconds can let a charging state from before the stop end it, or keep it after a start made in the
  Easee app.
- A stop Easee accepts but never applies leaves the charger charging while every `set_current` is only stored, so an
  energy manager cannot lower it until a start, a finished session or a new change into charging. A refused stop
  keeps the mark too, because a call that timed out may still have reached Easee.
- The session-finished states are `IsSessionFinished` without offline: completed, error, awaiting authentication and
  de-authenticating end the stop as well. Offline was left out because a connectivity blip (issue 14) would otherwise
  lift the stop.
- Energy guard pauses with `cmd.charge.stop` too, so its pause holds the same way and only its `cmd.charge.start`
  resumes the charger.

Reproduce: `cmd.charge.stop`, then `cmd.current_session.set_current 16`: nothing is sent; start charging from the
Easee app: the next value is written.

Closes when measured on a charger (clock skew between hub and Easee timestamps; a charger that ignores a stop) and the
operator confirms the list of states that end a user stop.
