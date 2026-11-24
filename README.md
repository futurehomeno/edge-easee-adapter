# Edge Easee adapter
[![Coverage Status](https://coveralls.io/repos/github/futurehomeno/edge-easee-adapter/badge.svg?branch=main&t=irAXge)](https://coveralls.io/github/futurehomeno/edge-easee-adapter?branch=main)

An edge-app adapter for Easee Home EV charger.

## Development utilities
* Easee charger documentation (https://developer.easee.cloud/docs)
* Easee API documentation (https://developer.easee.cloud/reference)

### Some of the useful messages:  

#### Login
Topic:  `/pt:j1/mt:cmd/rt:ad/rn:easee/ad:1`  
Message:
```json =
{
  "corid": null,
  "ctime": "2023-09-19T09:30:38.495926Z",
  "props": {},
  "resp_to": "pt:j1/mt:rsp/rt:cloud/rn:remote-client/ad:smarthome-app",
  "serv": "easee",
  "src": "smarthome-app",
  "tags": [],
  "type": "cmd.auth.login",
  "uid": "f61abfbc-8fcf-47c7-945d-9746c9d8ed1b",
  "val": {
    "username": "username",
    "password": "password",
    "encrypted": true
  },
  "val_t": "object",
  "ver": "1"
}
```
#### Start charging
Topic: `pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:10`

The optional `charging_mode` property selects the start current: `"slow"` starts at
`slowChargingCurrentInAmperes` when set. Every other start, with or without a mode, depends on the
session:

- The session start, before the charger has charged since the car was connected, starts at no less
  than `initial_charging_current`; with no current set yet, at the charger's max current.
- A resume, any later start in the session (after a stop or an energy manager's pause), starts at
  the current in force before the pause, never raised to `initial_charging_current`.
- A start after the adapter paused the charger for a current below `min_charging_current`, in the
  same session, starts at `initial_charging_current`, and not below `min_charging_current`.

A session ends when the charger reports disconnected, completed, error, offline, unknown, awaiting
authentication or de-authenticating; the next start is a session start again.

The charger's max current caps every start. A start without a mode keeps the current of a session
that was started slow.
```json =
{
"corid": null,
"ctime": "2023-09-20T10:48:32.687224Z",
"props": {},
"resp_to": "pt:j1/mt:rsp/rt:cloud/rn:remote-client/ad:smarthome-app",
"serv": "chargepoint",
"src": "smarthome-app",
"tags": [],
"type": "cmd.charge.start",
"uid": "4296187f-0347-4ebc-baad-8d4f2ccea52e",
"val": null,
"val_t": "null",
"ver": "1"
}
```

#### Logout
Topic: `pt:j1/mt:cmd/rt:ad/rn:easee/ad:1`
```json = 
{
"corid": null,
"ctime": "2023-09-20T10:49:32.129859Z",
"props": {},
"resp_to": "pt:j1/mt:rsp/rt:cloud/rn:remote-client/ad:smarthome-app",
"serv": "easee",
"src": "smarthome-app",
"tags": [],
"type": "cmd.auth.logout",
"uid": "e5e18917-8f22-4902-94c7-50552ab777b1",
"val": null,
"val_t": "string",
"ver": "1"
}
```

#### Stop charging
Topic: `pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1`
```json =
{
"corid": null,
"ctime": "2023-09-20T11:46:13.040817Z",
"props": {},
"resp_to": "pt:j1/mt:rsp/rt:cloud/rn:remote-client/ad:smarthome-app",
"serv": "chargepoint",
"src": "smarthome-app",
"tags": [],
"type": "cmd.charge.stop",
"uid": "0bc3b8fd-605c-457f-8f55-5907465adfd7",
"val": null,
"val_t": "null",
"ver": "1"
}
```


## Mockery:
go install github.com/vektra/mockery/v2@v2.53.5
