## Why

Review of PR #180 (release 3.2):
- The stored-password re-login fires only within 1 h of the session end, but refreshes come ~24 h
  apart, so it almost never fires ahead of the end.
- A start or stop sent at once is not waited out by Teardown, so the re-sync it arms once Easee
  answers can fire after the thing is deleted.

## What Changes

- Re-login instead of refreshing when the session ends within 25 h (longer than the 24 h access token).
- Teardown waits for an immediate send in flight, as it already does for a deferred one.
- Teardown refuses commands arriving after it (the router runs messages concurrently).
