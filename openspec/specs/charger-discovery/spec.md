# Charger Discovery and Sync Specification

## Purpose
Decide which of the account's Easee chargers this hub owns as FIMP things, keep that selection and
the things in step, and survive partial or empty responses from the Easee `/api/chargers` endpoint
without destroying devices.

## Requirements

### Requirement: Charger Selection Model
The selection SHALL be persisted as `selected_devices`. A nil selection means "every charger" and an
empty one means "no chargers"; the distinction separates an install that was never configured from
one where the user deselected everything. Configurations written before v3.0 carry no such key and
therefore read as nil.

#### Scenario: legacy configuration
- **WHEN** a configuration without `selected_devices` is loaded
- **THEN** the selection reads as nil and includes every charger

#### Scenario: user deselects everything
- **WHEN** the user saves an empty selection
- **THEN** the empty selection is persisted and no charger is included

#### Scenario: selection is copied on read and write
- **WHEN** the selection is read from or written to the config service
- **THEN** a clone that preserves the nil/empty distinction is used, rather than a plain slice copy

### Requirement: Manifest Device Selector
`GetManifest` SHALL populate the `configuration`/`selected_devices` selector block from a live
`/api/chargers` fetch. The block SHALL be marked ready only when the lifecycle connection state is
connected and the auth state is authenticated. Each option SHALL be labelled `<name> (<id>)`, falling
back to the bare ID when the charger has no name. A failure to prepare the selector SHALL be logged
and the manifest still returned.

#### Scenario: not yet logged in
- **WHEN** the manifest is requested while not connected or not authenticated
- **THEN** the block is not ready and carries the "please return to the previous page and log in"
  warning

#### Scenario: charger fetch fails
- **WHEN** the chargers fetch fails while preparing the selector
- **THEN** the error is logged and the manifest is returned without aborting

#### Scenario: unnamed charger
- **WHEN** a charger has an empty name
- **THEN** its option label is the charger ID alone

### Requirement: Configure Validates Before Mutating
`Configure` SHALL fetch the charger list first and reject a selection referencing IDs absent from it,
before any state is changed. An empty selection SHALL pass validation unchanged.

#### Scenario: unknown ID submitted
- **WHEN** the submitted selection names an ID Easee does not list
- **THEN** `Configure` returns an `unknown device IDs` error and nothing is persisted

#### Scenario: charger fetch fails during configure
- **WHEN** the chargers fetch fails
- **THEN** `Configure` returns a `fetch available chargers` error

### Requirement: Auto-Selection Cap
An unconfigured (nil) selection SHALL be materialised into a concrete list of charger IDs, capped at
10. Materialising it is what makes `cmd.thing.delete` stick, since "every charger" cannot express an
exclusion and the next sync would recreate the deleted thing. Chargers with an empty ID SHALL be
skipped. When more chargers exist than the cap, a warning naming the auto-selected IDs SHALL be
logged.

#### Scenario: fewer chargers than the cap
- **WHEN** the selection is nil and the account lists 3 chargers
- **THEN** all 3 IDs are materialised into the selection

#### Scenario: more chargers than the cap
- **WHEN** the selection is nil and the account lists 40 chargers
- **THEN** the first 10 IDs are selected and a warning is logged

#### Scenario: explicit selection is untouched
- **WHEN** the selection is not nil
- **THEN** it is returned unchanged regardless of the cap

### Requirement: Adopting An Existing Seeded Selection
When the selection is nil but the adapter already holds things, the IDs of those things SHALL be
adopted as the selection, so an upgrade from a version without a selection does not fall back to
"all" or to the auto-cap. Adoption SHALL run at initialize time, and again on login for a hub that
was upgraded while logged out and therefore never reached the boot-time adoption.

#### Scenario: upgrade with seeded things
- **WHEN** initialization finds a nil selection and existing things
- **THEN** the IDs of those things are persisted as the selection

#### Scenario: adoption write fails
- **WHEN** persisting the adopted selection fails at initialize time
- **THEN** a warning is logged and initialization continues

#### Scenario: no things held
- **WHEN** the selection is nil and the adapter holds no things
- **THEN** adoption is skipped

### Requirement: Re-Seeding An Authenticated Install With No Things
A login that authenticates and then fails to configure leaves the credentials stored and no things,
and nothing else ever re-seeds them: configuring the chargers has no caller but login, and the
periodic check is a no-op. Initialization SHALL therefore re-seed the selected chargers when
credentials are present but the adapter holds no things. A failure there SHALL be logged and SHALL
NOT fail initialization, and SHALL NOT consume the missing-charger retry budget, which belongs to a
login attempt.

#### Scenario: credentials without things
- **WHEN** initialization finds stored credentials but no things
- **THEN** the selected chargers are seeded rather than left absent until the next manual login

#### Scenario: re-seed fails
- **WHEN** the re-seed fails, for instance because the charger list cannot be fetched
- **THEN** a warning is logged, the retry budget is reset, and initialization continues

### Requirement: Stale ID Filtering On Adoption
Adoption at configure time SHALL restrict the owned IDs to those the account still lists, so a stale
ID cannot be carried forward. When nothing owned is listed at all — a different Easee account rather
than a short list — the owned IDs SHALL NOT be adopted, leaving the still-unconfigured selection to
be materialised by the auto-selection cap instead.

#### Scenario: one owned charger vanished from the account
- **WHEN** the hub owns two chargers and the account lists only one of them
- **THEN** only the listed one is adopted into the selection

#### Scenario: account holds none of the owned chargers
- **WHEN** the account lists chargers but none of the owned ones
- **THEN** adoption is skipped and the still-unconfigured selection falls through to the
  auto-selection cap, which picks and persists the first 10 chargers the new account lists

### Requirement: Missing Selected Charger Retry Budget
When selected chargers are absent from the `/api/chargers` response, seeding SHALL be refused and an
error returned for up to 3 consecutive attempts, because the sync destroys every persisted thing
absent from the seeds. On the 4th attempt the adapter SHALL log a warning, seed without them, and
remove them from the persisted selection so the budget is not spent again on every later login. The
budget SHALL reset when the set of missing IDs changes, compared as a sorted list so a reordering is
not mistaken for a new set.

#### Scenario: transient partial response
- **WHEN** a selected charger is missing from the response for the first time
- **THEN** an error naming the missing IDs and the retry count is returned and nothing is seeded

#### Scenario: charger still missing after the budget
- **WHEN** the same charger is missing on the 4th consecutive attempt
- **THEN** a warning is logged, the charger is dropped from the persisted selection, and the
  remaining chargers are seeded

#### Scenario: a different charger goes missing
- **WHEN** the missing set changes between attempts
- **THEN** the retry counter restarts from zero for the new set

#### Scenario: response becomes complete
- **WHEN** a later response lists every selected charger
- **THEN** the retry counter and the remembered missing set are cleared

### Requirement: Empty Charger List Handling
An empty `/api/chargers` response combined with a nil selection SHALL be routed through the same
retry budget as a partial response, by substituting the owned charger IDs as the selection. A
successful fetch listing nothing is treated as no more trustworthy than one listing too little.

#### Scenario: empty response with things held
- **WHEN** the chargers list comes back empty, the selection is nil, and the hub owns things
- **THEN** the owned IDs become the working selection and the retry budget applies

### Requirement: Thing Synchronisation
Selected chargers SHALL be synchronised into things by the framework sync, seeded with the charger ID
and the product string from `/api/chargers/{id}/details`. Details SHALL be fetched up front for every
selected charger, because the seed function cannot fail. A details fetch failure SHALL abort the
whole apply with a `fetch charger details` error. IDs the sync excludes SHALL be removed from the
selection, or the same vanished charger is re-announced on every later sync.

#### Scenario: details fetch fails
- **WHEN** fetching details for any selected charger fails
- **THEN** the apply returns a `fetch charger details` error and no sync runs

#### Scenario: sync excludes a charger
- **WHEN** the sync returns excluded IDs
- **THEN** those IDs are removed from the persisted selection

#### Scenario: seeds produced
- **WHEN** the sync returns at least one seed
- **THEN** the SignalR client is started

#### Scenario: sync partially fails
- **WHEN** the framework sync returns an error alongside its seeds and excluded IDs
- **THEN** the failure is logged, the excluded IDs are still removed from the selection, the SignalR
  client is still started, and the `sync things` error is still returned to the caller

### Requirement: Thing Composition
Each charger thing SHALL expose four services in group `ch_0`: chargepoint, electricity meter,
parameters and alarm_system. Its inclusion report SHALL carry `Easee` as manufacturer, `cloud` as
communication technology, `ac` as power source, a wake-up interval of `-1`, the charger ID as device
ID, and a product hash of `Easee - Easee - <product>`.

#### Scenario: thing created
- **WHEN** a charger thing is created
- **THEN** it exposes chargepoint, meter_elec, parameters and alarm_system services in group `ch_0`

#### Scenario: inclusion report identity
- **WHEN** the inclusion report is published
- **THEN** the device ID is the Easee charger ID and the manufacturer is `Easee`

### Requirement: Thing State Refresh At Creation
Thing creation SHALL refresh the charger state from `/api/chargers/{id}/config` and
`/api/chargers/{id}/site`, persisting the refreshed state. When the refresh fails with
`ErrNotLoggedIn` the thing SHALL be created from the stored state instead and the state SHALL NOT be
persisted, so a stored state that failed to load is not overwritten with zeros. When a refresh fails
for any other reason but the persisted state already holds the data that refresh would supply
(`IsConfigUpdateNeeded` / `IsSiteUpdateNeeded` returns false), the error SHALL be suppressed and the
stored data kept. Any other refresh error SHALL abort creation of the thing.

#### Scenario: not logged in at boot
- **WHEN** the state refresh fails with `ErrNotLoggedIn`
- **THEN** a warning is logged, the thing is created from the stored state, and the state is not
  written back

#### Scenario: other refresh failure with no stored data
- **WHEN** the state refresh fails for any other reason and the state still needs that update
- **THEN** thing creation returns the error

#### Scenario: other refresh failure with stored data
- **WHEN** the state refresh fails for any other reason but the stored state already holds the data
- **THEN** the error is suppressed and the thing is created from the stored state

#### Scenario: stored state unreadable
- **WHEN** the persisted state cannot be decoded
- **THEN** a warning is logged and an empty state is used as the starting point

### Requirement: Uninstall
`Uninstall` SHALL destroy all things, reset the configuration, clear the credentials and reset the
charging-session storage — sessions are keyed by charger ID alone, so a reinstall against a
different Easee account with a colliding ID would otherwise serve the previous account's data —
running
every step even when an earlier one fails and joining the errors. The app SHALL be marked not
configured regardless of those errors.

#### Scenario: config reset fails
- **WHEN** resetting the configuration fails
- **THEN** the credentials are still cleared, the joined error is returned, and the app is marked
  not configured
