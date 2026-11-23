# 33: a user stop is lifted by any later offered current

Found while adding `min_charging_current` (`configurable-minimum-current`).

Easee has one knob for both: `pause_charging` sets the dynamic current to 0, and any later dynamic-current write
above 0 resumes the charger. So after a user `cmd.charge.stop`, the next `cmd.current_session.set_current` that is not
skipped by the dedup resumes the charger. This is today's behaviour and the spec states it ("The Last Command Always
Wins": a balancing refresh cancels a pause). The new feature does not make it worse: its only resume step is
bypassing the dedup for the adapter's own sub-minimum pause, and `cmd.charge.stop` clears that memory.

The decided rule "never resume over a user stop that is in force" can therefore not be met on Easee without blocking
every offered-current write after a user stop. Not done here: it reverses a documented rule and its tests, and an
in-memory "user stopped" flag has no clear end (car unplugged? new session? Easee app start? adapter restart) - kept
too long it stops a charger from ever charging under an energy manager.

Reproduce: `cmd.charge.stop`, wait `offered_current_wait_time`, `cmd.current_session.set_current` with a value
different from the last one: the charger resumes.

Closes when the operator decides: keep today's behaviour, or block offered-current writes after a user stop until
`cmd.charge.start` (and say what else ends the block).

Resolved (the operator, 2026-11-23): after a user `cmd.charge.stop`, no `cmd.current_session.set_current` starts
charging; only `cmd.charge.start` does. The stop is kept in the adapter's database with its time, so it survives a
restart, and also ends on disconnected (the car unplugged) or a change into charging observed after the stop
(`configurable-minimum-current`, requirement "A User Stop Holds Until A Start").
