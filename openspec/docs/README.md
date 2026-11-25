# Vendor documents

Copies of Easee documents the specs and issues rely on. All seven came in with PR #228
(2026-11-21). The Markdown files are exports of the Easee developer site (front matter
`updatedAt`, project index https://developer.easee.com/llms.txt).

| File | Source | Used for |
|---|---|---|
| `easee-amqp-commands.md` | https://developer.easee.com/docs/amqp-commands (named in issue 10; the file names only the project index) | command IDs and the `CommandResponse` payload (`wasAccepted`, `resultCode`): issue 10 |
| `easee-enumerations.md` | https://developer.easee.com/docs/enumerations (canonical link in the HTML copy) | charger op modes, output phase (110) and other observation values: phase-mode and charger-state mapping, archived issue 23, change `tn-2-phase-modes` |
| `easee-enumerations.html` | https://developer.easee.com/docs/enumerations (its `<link rel="canonical">`) | the same page as saved from the browser, kept next to the Markdown export |
| `easee-observation-ids.md` | https://developer.easee.com/docs/observation-ids (named in the design of change `it-phase-currents`) | observation IDs the adapter subscribes to, e.g. 182-185 phase currents, 42/43 authorization and remote start: issues 35 and 47 |
| `easee-set-phase-mode.md` | source unknown (an API reference page; it holds the OpenAPI entry `charger_set_phase_mode`, `POST /api/chargers/{chargerId}/commands/set_phase_mode`) | the phase-mode values 1/2/3 and the call the adapter makes: `phase-mode` spec |
| `easee-chargeup-installation-guide.pdf` | source unknown (Easee "Installer guide, Easee Charge Up", English, created 2024-11-18) | background on grid types (TN, IT, TT), phase counts and current range; no spec or issue cites it yet |
| `easee-home-charge-installation-guide-no.pdf` | source unknown (Easee "Installatørveiledning, Easee Home, Easee Charge", Norwegian, created 2024-10-29; the guide points to easee.com/manuals) | the same for Easee Home and Easee Charge; no spec or issue cites it yet |
