# relogin-with-stored-password

## Why

Easee ends every session 60 days after the password login. Refreshing gets new access tokens but
never moves the refresh token's `exp`, and the last access token is cut short to expire at that
same moment. When it passes, the framework authenticator concludes `refresh token expired`, logs
the app out and charging control stays down until the user logs in again.

Seen on site 446e6864 (hub 048e3d96bee16b03ae): logins on 2026-05-11 and 2026-07-14 were followed
by logouts on 2026-07-10 and 2026-09-12, 60 days to the minute; the second left the charger
uncontrolled for four days. Site 8ee88ca8 was logged out 60 days and 17 minutes after its login.
Before March 2026 the refresh token's expiry moved forward with every refresh, so this is a change
on Easee's side, and no refresh timing can avoid it.

## What Changes

- The credential store keeps the username and password from `cmd.auth.login` next to the tokens,
  in the same 0640 secrets file - the approach edge-zaptec-adapter already uses.
- When the refresh token is within an hour of its end, or Easee rejects a refresh, the token
  exchange logs in again with the stored password instead of ending the session.
- A password Easee rejects is forgotten after that one attempt, so the app logs out as it does
  today rather than locking the user's account with repeated failed logins.
- Installs that logged in before this change hold no password and keep today's behaviour until
  their next login.
- cliffhanger is bumped from v1.3.5 to v1.3.7 and the version to 3.2.0.

## Impact

- `src/internal/config/config.go`, `src/internal/config/credentials.go`,
  `src/internal/api/authenticator.go`, `src/go.mod`, `src/go.sum`, `Makefile`
- Spec: `cloud-authentication`
