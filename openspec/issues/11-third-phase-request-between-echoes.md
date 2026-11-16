# A third phase request between two echoes is dropped

Found in the review of PR #214 (change `phase-set-before-echo`). Read from the code, not seen in a
log. This is the same gap as issue 9.

Say auto and then single phase both go out. The auto echo is newer than the observation the
single-phase target replaced, so the cached target goes back to auto. A three-phase request made
before the single-phase echo lands is then compared with auto and dropped as a no-op. When the
single-phase echo lands, the charger stays in single phase even though three-phase was the last
request. The report then stops honouring that request (see 9), so energy guard may correct the mode
on a later tick. Hitting this takes three requests inside the gap between two echoes.

Open question: can this be accepted, or does it need a pending target that is cleared by its
matching echo? The design of `phase-set-before-echo` rejected that as a second source of the
internal mode.
