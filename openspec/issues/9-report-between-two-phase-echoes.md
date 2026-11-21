# The report can flip between the echoes of two quick phase sets

Found while fixing the dropped phase set (change `phase-set-before-echo`). Read from the code, not
seen in a log.

Now that both sets of auto-then-single go out, the charger echoes auto and then single phase. The
auto echo is newer than the single-phase request, so `requestStillHolds` stops honouring the request
until the single-phase echo lands, and a phase-mode report in that gap can fall back to the output
phase or a refetched state and name three phases. Energy guard answering that report would send
single phase again, which bounces a charging session once more.

v3.3.0 review, checked with a test for this order (three-phase, then single): while a session runs or
is paused with its single leg cached, the report in the gap stays on that leg (the auto mode covers
it, so `outputPhaseStale` keeps it), and energy guard sees no change to answer. The flip needs a
charger with no usable cached leg, which is idle, so the repeated set bounces nothing.

Open question: is the gap short enough to ignore, or should a request outrank an internal-mode
observation that matches a target sent before it?
