# Tasks

## 1. Store the password

- [x] 1.1 Failing test: `Login` stores the username and password with the tokens
- [x] 1.2 Failing test: `RefreshCredentials` keeps the username and password across a rotation
- [x] 1.3 Add `Username`/`Password` to `Credentials`, store them on login, carry them through a refresh
- [x] 1.4 Tests green

## 2. Log in again when the session ends

- [x] 2.1 Failing test: access and refresh tokens expired with a password stored logs in again, no auth loss
- [x] 2.2 Failing test: a refresh token within an hour of its end logs in instead of refreshing
- [x] 2.3 Failing test: a refresh rejected with 401 logs in again
- [x] 2.4 Failing test: a password rejected with 400 is forgotten and the next call ends in auth loss without a second login
- [x] 2.5 Failing test: a login failing with 500 keeps the password and is retried
- [x] 2.6 Implement the re-login in the token exchange
- [x] 2.7 Tests green, `TestExpiredRefreshTokenLogsOut` unchanged

## 3. cliffhanger v1.3.7 and version

- [x] 3.1 Bump `github.com/futurehomeno/cliffhanger` to v1.3.7, `go mod tidy`
- [x] 3.2 Version 3.2.0

## 4. Gate

- [x] 4.1 `go build ./...`, `go vet ./...`, `golangci-lint run`
- [x] 4.2 `make test`
- [x] 4.3 `openspec validate relogin-with-stored-password --strict`
- [x] 4.4 Archive in the PR
