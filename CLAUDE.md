# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Stack

REST API for PhonkersBase, built with Go 1.26, Gin, and PostgreSQL 17 (via `pgx/v5`). Migrations run via `tern` and are embedded in the binary, applied automatically on startup. Logging via `zerolog`. Metrics via OpenTelemetry, pushed over OTLP to Grafana Cloud.

## Commands

```bash
# Local dev (hot reload via air, bind-mounted source)
docker compose -f docker-compose.dev.yaml up

# Build
go build ./...

# Run all tests
go test ./...

# Run a single test
go test ./internal/repository/ -run TestName

# Vet
go vet ./...
```

There is no separate lint step in CI beyond `go vet`.

## Architecture

Layered, single binary, entry point `cmd/api/main.go` → `internal/server.Run()`.

- `internal/server/server.go` — wires everything together: loads config, connects to Postgres, runs migrations, builds the `Handler` via `handlers.NewHandler(...)` with all repos injected, sets up the Gin router and routes, and handles graceful shutdown on `SIGINT`.
- `internal/config/config.go` — typed config loaded from env vars (`DB_URL`, `JWT_SECRET`, `CORS_ORIGIN`, `PORT`), validated via `go-playground/validator`. Fails fast on invalid config.
- `internal/handlers/` — Gin handlers, one file per resource (artist, label, source, suggestion, feedback, auth). All share a single `Handler` struct holding repo dependencies and `jwtSecret`.
- `internal/repository/` — Postgres data access via raw SQL with `pgx/v5`, one file per resource. `ErrNotFound` (defined in `repository/artist.go`) is the sentinel translated to HTTP 404 by handlers.
- `internal/domain/models.go` — domain types and DTOs (request inputs, list params) shared between handlers and repositories.
- `internal/middlewares/` — `auth.go` (JWT auth + `RequireRole`), `ratelimit.go` (per-IP token-bucket rate limiting, 50 req/min, defined as `RequestsPerMinute`).
- `internal/migrations/` — SQL migration files embedded via `embed.go`, run automatically against `public.schema_version` on startup.
- `internal/metrics/` — OpenTelemetry metrics, static package-level instruments initialized once via `metrics.Init()` in `server.Run()` (same pattern as the zerolog global logger). `metrics.Init` is a no-op (instruments bind to the SDK's default no-op provider) unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set, so local dev needs no collector. Three pieces:
  - `metrics.RecordHTTPRequest` — called from the `logger` middleware in `server.go`, records `http.server.request.duration` labeled by method/`c.FullPath()`/status.
  - `metrics.DBTracer` — a `pgx.QueryTracer` set on `pgxpool.ParseConfig(...).ConnConfig.Tracer` in `initDB`, records `db.client.query.duration` labeled by SQL verb (not full query text) and error.
  - `metrics.ErrorHook` — a `zerolog.Hook` attached to `log.Logger` that increments the `app.errors` counter on every `Error`/`Fatal` log event, so error metrics never need separate instrumentation at the call site.
  - Go runtime metrics (GC, heap, goroutines) are wired in automatically via `go.opentelemetry.io/contrib/instrumentation/runtime`.

### Routing structure (in `server.go`)

Routes are grouped under `/api/v1` with three tiers:
1. **Public** — no auth (artist/label/source listing, login, suggestion/feedback submission).
2. **Protected** (`AuthMiddleware`) — any valid JWT (logout, me, artist admin CRUD + stats, label CRUD, suggestion admin review, feedback admin review).
3. **Admin-only** (`AuthMiddleware` + `RequireRole("ADMIN")`) — user management under `/auth`, plus source CRUD.

`router.SetTrustedProxies(nil)` is intentional: no reverse proxy sits in front of this service, so `X-Forwarded-For` must not be trusted for rate-limiting `ClientIP()`.

## Database

- **artists** — core entity with Spotify metadata, UA/EN descriptions, free-text `notes`, and a `total_priority` column auto-recalculated by a trigger on `artist_labels` insert/update. Indexed on `(total_priority DESC, updated_at DESC, id ASC)` and `(updated_at DESC, id ASC)` for the default/secondary admin sort orders.
- **labels** — reference table; carries a `priority` weight (label names: `approved`, `blocked`, `warning`, `unknown`, `pride`, `base`).
- **artist_labels** — many-to-many join between artists and labels.
- **artist_countries** — per-artist country codes (`code`, ordered by `position`), each optionally linked to an `evidence_sources` row via `source_id`. There is no standalone `countries` reference table — countries are plain ISO codes, not rows with names.
- **evidence_sources** — bilingual (`name_uk`, `name_en`) catalog of how a country was determined for an artist; full CRUD via `/source`.
- **suggestions**, **feedbacks** — public-submission tables for new-artist suggestions and user feedback, with admin-only review/listing endpoints.

Schema is defined in `internal/migrations/00001_create_tables.sql`; admin user setup, `users`/`suggestions`/`feedbacks`/`evidence_sources`/`artist_countries` tables, and the `countries`→`artist_countries` migration live in `00002_admin_setup.sql`. New migrations follow the `NNNNN_description.sql` naming convention and are picked up automatically via `embed.go`.

## Containers

- `Dockerfile.dev` — Alpine + `air` for hot reload; source bind-mounted.
- `Dockerfile.prod` — multi-stage: `golang:1.26.2-alpine` builder (CGO-disabled static binary) → `alpine:latest` runtime, runs as non-root `app` user.
- `docker-compose.dev.yaml` — local API + Postgres 17 with healthchecks.
- `docker-compose.prod.yaml` — production compose, deployed via the `CD` GitHub Actions workflow over SSH. Passes through `OTEL_EXPORTER_OTLP_ENDPOINT`/`_PROTOCOL`/`_HEADERS` from the server's `.env` (not in this repo — set manually alongside `JWT_SECRET`).

## CI/CD

- **CI** (`.github/workflows/ci.yaml`): on push/PR to `main` — `go mod download`, `go build ./...`, `go test ./...`, `go vet ./...`.
- **CD** (`.github/workflows/cd.yaml`): triggers after CI succeeds on `main`. Detects whether API code (`internal/`, `cmd/`, `go.mod`, `go.sum`, `Dockerfile.prod`) or `docker-compose.prod.yaml` changed (by diffing against the currently deployed image's revision label) and only rebuilds/redeploys when needed. Images tagged with `latest` and the commit SHA; deploy pulls and restarts via `docker compose` over SSH.
