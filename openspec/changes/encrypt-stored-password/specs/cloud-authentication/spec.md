## ADDED Requirements

### Requirement: Stored Password Encrypted
The credential store SHALL write the username and password to the secrets file encrypted with
AES-256-GCM under a key built into the adapter, and SHALL read back a plain-text value written by an
earlier version. A value that does not decrypt SHALL read as empty without failing the load.

#### Scenario: Secrets file holds no plain password
- **WHEN** credentials with a username and password are saved
- **THEN** the secrets file contains neither in plain text and loading it returns both unchanged

#### Scenario: Plain-text password from an earlier version
- **WHEN** the secrets file holds a username and password in plain text
- **THEN** loading it returns both unchanged

#### Scenario: Password that does not decrypt
- **WHEN** the stored password cannot be decrypted
- **THEN** loading keeps the tokens and reads the password as empty

#### Scenario: Plain-text secrets sealed at startup
- **WHEN** the adapter starts with a secrets file holding a plain-text username or password
- **THEN** it rewrites the file encrypted and removes the backup that holds the plain text
