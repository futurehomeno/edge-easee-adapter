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
