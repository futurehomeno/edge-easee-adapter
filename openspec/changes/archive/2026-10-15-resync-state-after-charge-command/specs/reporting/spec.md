## MODIFIED Requirements

### Requirement: Charger State Report
`evt.chargepoint.state_report` SHALL report the cached charger state, except that a cached total
power greater than zero SHALL be reported as `charging` when its observation is not older than the
cached charger-state observation. A charger-state observation newer than the last power reading
SHALL win, so a power value the stream stopped updating cannot hold the charger at `charging`. The
chargepoint service SHALL advertise in `sup_states` the FIMP states `unknown`, `disconnected`,
`ready_to_charge`, `charging`, `finished`, `error`, `suspended_by_ev` and `requesting` — Easee's nine
operating states mapped to FIMP and deduplicated, `offline` and `de-authenticating` both collapsing
to `unknown`. The FIMP names are not the Easee ones: Easee's awaiting start maps to
`ready_to_charge`, while Easee's ready to charge maps to `suspended_by_ev`.

#### Scenario: power observed while state says otherwise
- **WHEN** the cached total power is above zero and its observation is not older than the cached state
- **THEN** `charging` is reported

#### Scenario: newer state after the last power reading
- **WHEN** the cached total power is above zero but a non-charging state was observed after it
- **THEN** the cached charger state is reported

#### Scenario: no power observed
- **WHEN** the cached total power is zero or unset
- **THEN** the cached charger state is reported as-is

#### Scenario: supported states advertised
- **WHEN** the chargepoint service specification is built
- **THEN** each distinct FIMP state appears once in `sup_states`
