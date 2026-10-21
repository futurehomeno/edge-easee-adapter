# Reporting

## ADDED Requirements

### Requirement: Connectivity Reported On Change
The adapter SHALL send a charger's node report as soon as its connectivity changes, without waiting
for the periodic connectivity task: when its SignalR subscribe succeeds or fails, when the SignalR
client disconnects, and when the charger's cloud-connected or offline state flips. The report SHALL
be non-forced, so an unchanged status is deduped by the reporting cache.

#### Scenario: login after a logout
- **WHEN** the SignalR client reconnects after a login and the charger resubscribes
- **THEN** a node report with connectivity up is sent right away

#### Scenario: SignalR connection lost
- **WHEN** the SignalR client disconnects
- **THEN** a node report with connectivity down is sent for every registered charger

#### Scenario: charger leaves the cloud
- **WHEN** the charger's cloud-connected observation flips to false or back to true
- **THEN** a node report is sent with the new connectivity
