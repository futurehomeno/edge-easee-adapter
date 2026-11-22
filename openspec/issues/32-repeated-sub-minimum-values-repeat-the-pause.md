# 32: every sub-minimum offered current sends pause_charging again

Found while adding `min_charging_current` (`configurable-minimum-current`).

A sub-minimum value is not deduplicated: an energy manager resending 0 A on every balancing tick (about every 38 s,
issue 3) sends `pause_charging` once per `offered_current_wait_time` window, each followed by a re-sync. Chosen
because it fails safe - the charger stays paused even if something else resumed it - and it mirrors issue 3, where an
unchanged current is rewritten once the window passes.

Unknown: what Easee answers to `pause_charging` on a charger already paused. If it is an error, the routing divert
leaves the message to cliffhanger, which then answers with its own "at least 6A" error.

Also changed alongside: `initial_charging_current` had no FIMP route on Easee; it now has get/set routes and a setter
(6–32 A, never below the minimum). The test configs under `testdata/` carry neither current setting, as before.

Closes when measured on a charger (answer to a repeated `pause_charging`) and the operator decides whether repeats
within the adapter's own pause should be skipped.
