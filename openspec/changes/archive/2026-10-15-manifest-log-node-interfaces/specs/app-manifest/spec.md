## ADDED Requirements

### Requirement: Routed Interfaces Declared
The packaged app manifest SHALL declare both sides of the log-level and single-node interfaces the
adapter routes, with commands as `in` and events as `out`: `cmd.log.get_level` and
`cmd.log.set_level` with `evt.log.level_report`, and `cmd.network.get_node` with
`evt.network.node_report`.

#### Scenario: manifest lists the pairs
- **WHEN** a client reads the packaged manifest
- **THEN** it finds all five message types among the service interfaces, each with its direction
