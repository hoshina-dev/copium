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
- **Web UI**: a Vite + React + Mantine SPA in `webui/` is built into `webui/dist`
  and embedded into the Go binary via `go:embed`. Fiber serves the SPA on `/`
  with an `index.html` fallback for client-side routing, while the JSON API
  stays under `/api/v1/*`.

> **Heads up:** copium ships with **no auth**. Don't expose it to the public
> internet - put it behind your usual ingress / SSO if you must.

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
| `make webui-install` | `npm install` inside `webui/` |
| `make webui-dev` | hot-reloading Vite on `http://localhost:5173`, proxies `/api`, `/swagger`, `/scalar` to `:8081` |
| `make webui-build` | one-shot Vite production build into `webui/dist` |
| `make build` | builds the SPA, then the Go binaries (web UI is embedded in `bin/copium`) |

## Configuration

Copy `.env.example` to `.env` and adjust. See the file for every supported variable.

## Observability

Copium is built for production operability. Logs, metrics, and traces are all
emitted through the standard OpenTelemetry APIs, and the behaviour switches on
a single env flag.

### Signals

| signal | when otel disabled | when `OTEL_ENABLED=true` |
| --- | --- | --- |
| **Logs** (`log/slog`) | Text handler → stdout | Tee: stdout **and** OTLP/gRPC via the `otelslog` bridge |
| **Metrics** (`otel.Meter`) | Noop meter (free) | OTLP/gRPC periodic reader |
| **Traces** (`otel.Tracer` + `otelfiber`) | Noop tracer | OTLP/gRPC batch exporter |

All three share the same `OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_EXPORTER_OTLP_INSECURE`
settings and the `OTEL_SERVICE_NAME` resource attribute.

### What gets logged

- **HTTP access log** (Fiber `logger` middleware) — one line per request with
  timestamp, status, method, path, latency, and error. `/healthz` / `/readyz`
  are skipped so probes don't flood the log.
- **Send lifecycle** — `email.queued` with `outbox_id`, `template_code`,
  `version`, `to`, `from`.
- **Worker dispatch** — `worker.claimed` (batch summary), then per row either
  `worker.send_ok` (with `provider`, `provider_msg_id`) or `worker.send_failed`
  (with `attempt`, `error`, `retry_at`). Mark-sent/mark-fail DB errors surface
  as `worker.mark_*_error`.
- **Lifecycle** — `server.listening`, `server.shutting_down`, `worker.enabled`,
  `email.sender.configured`.

All of these are structured key/value `slog` records, so they ship to OTLP as
proper log records (not dumped strings) when otel is on.

### What gets measured

The HTTP layer and worker publish these instruments out of the box:

| metric | kind | attributes | meaning |
| --- | --- | --- | --- |
| `http.server.request.duration` | histogram (ms) | `http.route`, `http.method`, `http.status_code` | Per-route latency — p50/p95/p99 via `histogram_quantile` |
| `http.server.active_requests` | up/down counter | `http.method` | In-flight requests |
| `http.server.request.size` / `response.size` | histograms (bytes) | as above | Request / response payload sizes |
| `copium.worker.claimed` | counter | — | Rows pulled from the outbox |
| `copium.worker.sent` | counter | `provider` | Successful deliveries |
| `copium.worker.failed` | counter | `provider` | Sender errors (will retry / dead-letter) |
| `copium.worker.dispatch.duration` | histogram (s) | `provider` | Time spent in `Sender.Send` |

The `http.server.*` metrics come from `otelfiber`, mounted unconditionally —
flipping `OTEL_ENABLED=true` is the only step needed to start shipping them.

Add more instruments in the same pattern — call `otel.Meter("copium/<area>")`
and the global MeterProvider routes through OTLP automatically.

## Web UI

The management UI is a single-page Vite + React + Mantine app. It lets you
manage email templates (create, version, set active), edit subject/body/JSON
Schema with a live preview, and dispatch test emails with a schema-driven form
that polls the outbox until the email is `sent` or `dead`.

Two ways to run it:

| mode | command | URL |
| --- | --- | --- |
| Hot reload | `make run` + `make webui-dev` (in two terminals) | `http://localhost:5173` |
| Embedded prod | `make build && ./bin/copium` | `http://localhost:8081/` |

In dev mode, Vite proxies `/api`, `/swagger`, `/scalar`, `/healthz` and
`/readyz` to the Go backend on `:8081`, so the same `fetch("/api/v1/...")`
calls work in both modes. In embedded mode, Fiber serves
`webui/dist/assets/*` and falls back to `index.html` for any path that isn't
the API or docs.

## API documentation

The server exposes two OpenAPI UIs out of the box:

| URL | UI |
| --- | --- |
| `http://localhost:8081/swagger/index.html` | Classic Swagger UI |
| `http://localhost:8081/scalar` | Scalar UI (modern, `data-url=/swagger/doc.json`) |
| `http://localhost:8081/swagger/doc.json` | Raw OpenAPI 2.0 spec (machine-readable) |

The committed `docs/` package is generated from swag annotations on handlers
plus the `@title`/`@BasePath` block in `cmd/server/main.go`. Regenerate after
changing any annotation:

```sh
make swagger        # writes docs/docs.go, docs/swagger.json, docs/swagger.yaml
```

The Make target fails if any model still leaks the Go import path
(`github_com_*`) so reviewers always see clean type names in the spec.

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
| Webui | `Webui: Dev (npm run dev)` | runs `make webui-dev` (Vite dev server on `:5173`) |
| Compound | `Dev Stack + Server` | one-click: Compose up then Run server |

Local IDE state (`workspace.xml`, `codeStyles/`, etc.) is intentionally
gitignored - only the run configs and `.idea/.gitignore` are shared.
