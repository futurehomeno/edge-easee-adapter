# The phase-mode spec does not describe what a no-op set records

Found while fixing the dropped phase set (change `phase-set-before-echo`).

`Applying A Phase Mode` said a set whose target equals the cached internal mode records no
requested mode. The code records the leg unless the charger is charging on a live output-phase leg,
and while it is, re-records it over an earlier request that still outranks that leg; `legToRecord`
can also substitute the known leg for the requested one. The test table case "already in the target
mode" expects the record. The change drops the wrong claim but does not specify the recording.

Open question: specify the no-op recording and the leg substitution in `phase-mode`, or leave them
to the code and its tests?
