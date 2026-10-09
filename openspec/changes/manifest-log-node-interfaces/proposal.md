# manifest-log-node-interfaces

## Why

PR #67 review: the packaged manifest lists `cmd.log.set_level` and `evt.network.node_report` but
not their counterparts `cmd.log.get_level`, `evt.log.level_report` and `cmd.network.get_node`, which cliffhanger's
debug and adapter routes serve.

## What Changes

- The packaged manifest declares `cmd.log.get_level`, `evt.log.level_report` and `cmd.network.get_node`.
