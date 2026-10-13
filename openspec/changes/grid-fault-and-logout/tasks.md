# Tasks

## 1. Keep the topology through a wiring fault (#164)

- [x] 1.1 Failing test: a zero-phase config does not overwrite the stored grid type and phases
- [x] 1.2 Failing test: a healthy reading still replaces them
- [x] 1.3 Skip the state write when `ToFimpGridType` yields zero phases
- [x] 1.4 Tests green

## 2. Logout verifies the store (#165)

- [x] 2.1 Failing test: a failed clear that leaves credentials does not mark the app logged out
- [x] 2.2 Failing test: a failed clear that leaves the store empty still logs out
- [x] 2.3 Check the store before marking the lifecycle
- [x] 2.4 Tests green

## 3. Gate

- [x] 3.1 `go build ./...`, `go vet ./...`, `golangci-lint run`
- [x] 3.2 `make test`
- [x] 3.3 `openspec validate grid-fault-and-logout --strict`
