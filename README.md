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

```sh
docker run -d --name copium-pg -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=copium postgres:16
docker run -d --name copium-mailhog -p 1025:1025 -p 8025:8025 mailhog/mailhog

# apply migrations (manual for now)
psql "postgres://postgres:postgres@localhost:5432/copium?sslmode=disable" -f sql/001_init.up.sql

EMAIL_PROVIDER=smtp SMTP_HOST=localhost SMTP_PORT=1025 make run
# Mailhog UI: http://localhost:8025
```
