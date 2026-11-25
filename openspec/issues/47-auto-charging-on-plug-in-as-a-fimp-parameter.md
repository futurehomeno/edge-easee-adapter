# 47: auto charging on plug-in could be a FIMP parameter, as on the FH charger

## What happened

A user stop holds until `cmd.charge.start`, an unplug, or charging seen after the stop (`configurable-minimum-current`,
issue 33). Whether a re-plugged car then starts by itself is decided by Easee's own charger settings, observations
42 AUTHORIZATION REQUIRED and 43 REMOTE START REQUIRED (`openspec/docs/easee-observation-ids.md:223,231`). The adapter
neither reads nor sets them: none is in `src/internal/model/model.go`, and the REST client only writes
`maxChargerCurrent`/`dynamicChargerCurrent` (`src/internal/api/http.go`).

The FH charger exposes the same choice as the FIMP parameter `free_charging` ("When enabled automatically starts
charging when cable is plugged in", zigbee-ad `descriptors/fh_charge.json`).

## Reproduce

With authorization and remote start not required on the charger, `cmd.charge.stop`, then unplug and plug the car
back in: the charger charges with no `cmd.charge.start`, and nothing on the hub can change that.

## What would close it

The operator decides whether to add the feature. If yes, a parameter shaped like `free_charging` (bool), reported
from observations 42/43 and set through the charger settings API. The setting names and the API call must be
checked against the Easee API documentation first.
