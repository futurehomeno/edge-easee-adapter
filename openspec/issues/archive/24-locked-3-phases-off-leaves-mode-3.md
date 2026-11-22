Resolved: 2026-11-22, the user: a setting changed during a session forces nothing; the new value is used by the next single-to-three-phase request and from the session end on. A session that started with the setting on is released at its end even when the setting is off by then. Known limits, in the design of `add-use-locked-3-phases`: the handler remembers the setting in memory, so an adapter restart between turning it off and the session end leaves mode 3; so does a lock placed by a hub request in a session that started with the setting off, when the setting is off again at its end. The text below is the issue as first narrowed in the PR 230 review, before that decision.

# use_locked_3_phases: turning it off releases the lock only at the next state change

Found while adding `use_locked_3_phases` (change `add-use-locked-3-phases`); narrowed in the PR 230 review.

Setting `use_locked_3_phases` to false does not write to the chargers. A charger on mode 3 gets auto at its
next charger state change (the disconnect at the latest): the handler remembers that the setting was on at
the previous change. Left:

- Until that state change the charger stays on mode 3, so a car stalled on the lock with a steady state is
  freed only by unplugging.
- The memory is in the handler: an adapter restart between turning the setting off and the next state change
  leaves mode 3, and the hub does not repeat a three-phase request the report already shows.

How to observe: setting on, a session on mode 3; set it to false and restart the adapter before the charger
state changes; disconnect the car. The charger config still shows phase mode 3.

Closes when: the user accepts this, or the setter pushes auto to every charger on mode 3 at once (the setter
needs a way to the chargers' cached mode: config service -> SignalR manager -> handlers).
