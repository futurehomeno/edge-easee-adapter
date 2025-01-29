# Edge Easee adapter
[![Build Status](https://app.travis-ci.com/futurehomeno/edge-easee-adapter.svg?token=BT5GpzawZfLuMbxzdzfx&branch=main)](https://app.travis-ci.com/futurehomeno/edge-easee-adapter)
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
