# connectivity-on-reconnect

## Why

The node report (`evt.network.node_report`) is only sent by the periodic connectivity task
(`polling_interval`, default 10 min). After a logout the chargers go "no signal"; after the login
SignalR reconnects and every charger resubscribes within a second, yet the app keeps showing
"no signal" for up to 10 min. Every SignalR drop/reconnect lags the same way.

## What Changes

- The charger's node report is sent (non-forced, deduped by the reporting cache) as soon as its
  connectivity changes: subscribe result, client disconnect, cloud-connected flip and the charger
  state going offline/online. The periodic task stays as the fallback.
