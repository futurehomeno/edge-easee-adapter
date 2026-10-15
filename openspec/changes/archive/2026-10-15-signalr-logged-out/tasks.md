# Tasks

- [x] 1 Failing test: not logged in warns nothing and a `Start()` after login dials at once
- [x] 2 Treat `auth.ErrNotLoggedIn` as a waiting state in `handleConnection`/`getConnection`; wake on `Start()`
- [x] 3 Tests green, `make lint`, `openspec validate signalr-logged-out --strict`
