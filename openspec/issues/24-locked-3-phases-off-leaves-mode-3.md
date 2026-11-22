# use_locked_3_phases: turning it off does not push auto

Found while adding `use_locked_3_phases` (change `add-use-locked-3-phases`).

Setting `use_locked_3_phases` to false changes only the mapping. A charger already locked to mode 3
stays there: the session-end rule is skipped with the setting off, and the hub does not repeat a
three-phase request the report already shows. It returns to auto only at the next three-phase
request (or a single-phase one).

How to observe: setting on, a session running on mode 3; set it to false; disconnect the car. The
charger config still shows phase mode 3.

Closes when: the user accepts this, or the setter pushes auto to every charger on mode 3 when the
setting goes off.
