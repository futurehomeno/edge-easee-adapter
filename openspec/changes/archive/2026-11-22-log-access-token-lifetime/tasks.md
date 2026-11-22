# Tasks

## 1. Log the lifetime
- [x] 1.1 Write a failing test asserting a login and a refresh each log
  `[auth] Access token expiresIn=3600s expiresAt=<JWT exp>` at info level
- [x] 1.2 Log the parsed `expiresIn` and the derived expiry in `api/http.go`, run the test green

## 2. Gate
- [x] 2.1 `go vet ./...`, `go build ./...`, `golangci-lint run`
- [x] 2.2 `make test`
- [x] 2.3 `openspec validate log-access-token-lifetime --strict`
