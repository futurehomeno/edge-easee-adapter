# signalr-logged-out

## Why

With chargers' things on the hub but no Easee session (logged out, or credentials cleared by an
auth loss), the SignalR client retries its start forever and warns
`Unable to start signalr client err: unable to get access token (signalR): not logged in` on every
attempt - 23k lines on 16 of 38 beta hubs (3.1.4, also on 2.8.0). A login then waits out the
backoff (up to ~11 min) before the client connects.

## What Changes

- "Not logged in" is a waiting state, not a failure: logged once at info per streak, then debug;
  no extra one-minute sleep.
- `Start()` on a running client wakes a start attempt waiting out its backoff, so the login path
  (which already calls `Start()`) connects at once. Logout still closes the client.
