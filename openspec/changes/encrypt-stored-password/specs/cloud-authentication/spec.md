## ADDED Requirements

### Requirement: Stored Password Encrypted
The credential store SHALL write the username and password to the secrets file encrypted with
AES-256-GCM under a key built into the adapter, and SHALL read back a plain-text value written by an
earlier version.

#### Scenario: Secrets file holds no plain password
- **WHEN** credentials with a username and password are saved
- **THEN** the secrets file contains neither in plain text and loading it returns both unchanged

#### Scenario: Plain-text password from an earlier version
- **WHEN** the secrets file holds a username and password in plain text
- **THEN** loading it returns both unchanged
