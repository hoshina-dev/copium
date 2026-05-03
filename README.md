# copium

Communication API. Manages versioned email templates and dispatches emails asynchronously
through a DB-backed outbox to a pluggable `Sender` (SMTP, SES, SendGrid, noop).

## Architecture

- **Fiber v2** HTTP server with optional **OpenTelemetry** instrumentation.
- **Layered**: `handlers` -> `services` -> `repositories` (GORM/Postgres).
- **Async dispatch**: `POST /api/v1/emails/send` enqueues into `email_outbox`; a background
  worker polls with `FOR UPDATE SKIP LOCKED`, renders the snapshot via the `Sender`,
  retries with exponential backoff, dead-letters at `max_attempts`.
- **Recipient resolution**: a `custapi` HTTP client looks up `user_id -> email`.
- **Template versioning**: every edit creates a new immutable `email_template_versions`
  row; the parent `email_templates.active_version_id` is the pointer used by send.
- **Param validation**: each version stores a JSON Schema; submitted params are
  validated against it before render.

## Dependency injection

Every collaborator crosses package boundaries as a **consumer-owned interface** (Go
accepted-interface idiom). Concrete adapters live in their own packages and are
wired exactly once in the composition root, `cmd/server/main.go`. `Clock` and `IDGen`
are also injected so tests are deterministic.

## TDD workflow

1. `make test-watch` (requires `gotestsum`) in a side terminal -> red/green feedback on save.
2. Write the failing test first, run `make test`, see it fail, implement the minimum,
   see it pass, refactor.
3. Mocks live in `mocks/` and are regenerated with `make mocks` (uses `.mockery.yaml`).

### Make targets

| target | what it does |
| --- | --- |
| `make test` | `go test -race -count=1 ./...` |
| `make test-unit` | excludes integration-tagged tests (no Docker) |
| `make test-watch` | continuous via `gotestsum --watch` |
| `make cover` | coverage profile + summary |
| `make lint` | `golangci-lint run ./...` |
| `make mocks` | regen testify mocks |
| `make run` | `go run ./cmd/server` |
| `make swagger` | regenerate OpenAPI docs |

## Configuration

Copy `.env.example` to `.env` and adjust. See the file for every supported variable.

## Local dev with Docker

The bundled compose file boots Postgres 18 (with migrations + seed auto-applied
on first init) and Mailhog:

```sh
docker compose -f docker-compose.dev.yml up -d
EMAIL_PROVIDER=smtp SMTP_HOST=localhost SMTP_PORT=1025 make run
# Mailhog UI: http://localhost:8025
```

## JetBrains run configurations

Shared run configs live under `.idea/runConfigurations/` and are picked up
automatically when the project is opened in GoLand or IntelliJ IDEA Ultimate
(Go plugin required). Available entries in the run dropdown:

| group | name | what it does |
| --- | --- | --- |
| Run | `Run server` | `go run ./cmd/server` with safe defaults (noop sender) |
| Run | `Run server (SMTP -> Mailhog)` | same but `EMAIL_PROVIDER=smtp` -> `localhost:1025` |
| Test | `Test: All` | `go test -race -count=1 ./...` |
| Test | `Test: Current Package` | runs tests in the package of the active editor file |
| Test | `Test: Integration (-tags=integration)` | spins testcontainers (10m timeout) |
| Test | `Test: All (Coverage)` | same as Test: All but with the GoLand coverage runner |
| Make | `Make: test` / `test-watch` / `mocks` / `lint` | invokes the Makefile targets |
| Docker | `Compose: Up Dev Stack` | brings up Postgres + Mailhog from `docker-compose.dev.yml` |
| Compound | `Dev Stack + Server` | one-click: Compose up then Run server |

Local IDE state (`workspace.xml`, `codeStyles/`, etc.) is intentionally
gitignored - only the run configs and `.idea/.gitignore` are shared.
