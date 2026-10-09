# encrypt-stored-password

## Why

Since 3.2.0 the Easee username and password sit in plain text in `data/secrets.json` (mode 0640),
so anyone who can read the file or a backup of it gets the user's Easee account.

## What Changes

- The username and password are written to `secrets.json` encrypted with AES-256-GCM under a key
  built into the binary. Tokens stay as they are.
- A plain-text value from 3.2.0/3.2.1 is still read, and written encrypted on the next save.
- The key is in the (public) source, so this keeps the password out of casual reads, logs and
  backups; it does not protect it from someone who has the file and the binary.
- Version 3.2.2.

## Impact

- `src/internal/config/config.go`, `src/internal/config/secret.go`, `src/internal/api/authenticator.go`,
  `src/internal/config/credentials.go`, `Makefile`
- Spec: `cloud-authentication`
