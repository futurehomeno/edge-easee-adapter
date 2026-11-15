# An unchanged offered current is rewritten once the wait window passes

Found while fixing the repeated clamp warning (change `clamp-warn-once`).

On one hub Energy Guard sent `set_current 20` 807 times in 8.5 hours (about every 38 s) while the
charger's max was 13 A; 556 of them became Easee writes of an unchanged 13 A.

`setOfferedCurrent` clamps first and then dedups, so the dedup compares the clamped value: it is not
comparing the pre-clamp 20 A. It skips a repeat only while `clock.Since(lastSet) <
offered_current_wait_time` (30 s), which is what the `Offered Current` requirement says. Requests
spaced more than 30 s apart therefore reach Easee every time, even when the value is unchanged.

Open question: is that rewrite meant as a refresh (for example in case Easee or the charger drops the
dynamic current), or should an unchanged value be skipped for longer, until the charger reports a
different offered current or a session starts? Neither the spec nor the code states the reason.
