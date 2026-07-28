# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Stack

REST API for PhonkersBase, built with Go 1.26, Gin, and PostgreSQL 17 (via `pgx/v5`). Migrations run via `tern` and are embedded in the binary, applied automatically on startup. Logging via `zerolog`. Metrics via OpenTelemetry, pushed over OTLP to Grafana Cloud.

## Commands

```bash
# Local dev (hot reload via air, bind-mounted source; API on :8080, Postgres on :5432)
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

There are currently no `_test.go` files in the repo — `go test ./...` runs clean by default, so it is not a meaningful check on its own. There is no separate lint step in CI beyond `go vet`.

## Architecture

Layered, single binary, entry point `cmd/api/main.go` → `internal/server.Run()`.

- `internal/server/server.go` — wires everything together: loads config, connects to Postgres, runs migrations, builds the `Handler` via `handlers.NewHandler(...)` with all repos injected, sets up the Gin router and routes, and handles graceful shutdown on `SIGINT`.
- `internal/config/config.go` — typed config loaded from env vars (`DB_URL`, `JWT_SECRET` (min 32 chars), `CORS_ORIGIN` (comma-separated, defaults to `http://localhost:3000`), `PORT` (default `8080`), `OTEL_SERVICE_NAME` (default `pb-api2`), `OTEL_EXPORTER_OTLP_ENDPOINT`), validated via `go-playground/validator`. Fails fast on invalid config. **Rule: `os.Getenv` only appears in this file.** Every other package receives config values as explicit parameters/struct fields from `cfg`, never reads env vars itself — this keeps env-var validation and defaulting in one place instead of scattered across the codebase.
- `internal/handlers/` — Gin handlers, one file per resource (artist, label, source, organisation, suggestion, feedback, history, auth). All share a single `Handler` struct (`handler.go`) holding repo dependencies and `jwtSecret`.
- `internal/repository/` — Postgres data access via raw SQL with `pgx/v5`, one file per resource. `ErrNotFound` (defined in `repository/artist.go`) is the sentinel translated to HTTP 404 by handlers. `like.go` holds `escapeLikePattern`, which every user-supplied search term must pass through so `%`/`_`/`\` match literally instead of acting as wildcards.
- `internal/domain/models.go` — domain types and DTOs (request inputs, list params) shared between handlers and repositories. `internal/domain/history.go` holds the change-history types plus the `EntityType*`/`ChangeAction*` constants and their validators.
- `internal/middlewares/` — `auth.go` (JWT auth + `RequireRole`), `ratelimit.go` (per-IP token-bucket rate limiting, 50 req/min, defined as `RequestsPerMinute`).
- `internal/migrations/` — SQL migration files embedded via `embed.go`, run automatically against `public.schema_version` on startup.
- `internal/metrics/` — OpenTelemetry metrics, static package-level instruments initialized once via `metrics.Init()` in `server.Run()` (same pattern as the zerolog global logger). `metrics.Init` is a no-op (instruments bind to the SDK's default no-op provider) unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set, so local dev needs no collector. Three pieces:
  - `metrics.RecordHTTPRequest` — called from the `logger` middleware in `server.go`, records `http.server.request.duration` labeled by method/`c.FullPath()`/status.
  - `metrics.DBTracer` — a `pgx.QueryTracer` set on `pgxpool.ParseConfig(...).ConnConfig.Tracer` in `initDB`, records `db.client.query.duration` labeled by SQL verb and query name (not full query text), and logs a `slow query` warning above `slowQueryThreshold` (200ms).
  - `metrics.ErrorHook` — a `zerolog.Hook` attached to `log.Logger` that increments the `app.errors` counter on every `Error`/`Fatal` log event, so error metrics never need separate instrumentation at the call site.
  - Go runtime metrics (GC, heap, goroutines) are wired in automatically via `go.opentelemetry.io/contrib/instrumentation/runtime`.

### Query naming convention

Every SQL string in `internal/repository/` starts with a `-- name: <resource>.<operation>` comment (e.g. `-- name: label.get_all`). `metrics.queryName` parses it into the low-cardinality `query.name` label used by the DB duration metric and the slow-query log; queries without it fall back to `unknown`. Add the comment to any new query.

### Artist search semantics

`buildSearchCondition` (`internal/repository/artist.go`) splits `search` on commas and branches on the
term count, and that branch is deliberate rather than incidental:

- **One term** → `buildPartialSearchCondition`: case-insensitive substring (`ILIKE '%term%'`) across name, the
  localized description and `SPLIT_PART(link, '?', 1)`. This is the UI search box, where the user types a fragment.
- **Two or more terms** → `buildExactSearchCondition`: exact matching only. Bare 22-char Spotify IDs and
  `open.spotify.com`/`spotify:artist:` links collect into `spotify_id = ANY($n)`; everything else is lowered in Go
  and collected into `lower(a.name) = ANY($m)`, served by the `idx_artists_lower_name` expression index (`00012`).

Two array params, whatever the term count. The reason is cost: substring-matching N terms emits 3N `ILIKE`
predicates that the planner resolves as a sequential scan (measured ~10 ms per term — a 46-term search took
509 ms and grows with both term count and table size), whereas the exact path is one bitmap index scan (~1.5 ms).
Multi-term search only ever comes from callers enumerating identifiers they already hold, so exact matching is
also the more correct semantic — substring matching lets a short term like `Muse` hit `MYSTICMUSE`.

Keep `lower()` on the column side of that comparison: `lower(a.name) = ANY(...)` matches the expression index,
`a.name = ANY(...)` would not. Public API behavior is documented in `README.md` under "Search semantics".

### Change history

Mutating handlers for artist, label, source, and organisation call `h.recordChange(c, entityType, entityID, entityName, action, old, new)` (`handlers/handler.go`) after a successful write. It is best-effort — a failed history insert logs but never fails the request — and captures the pre-mutation snapshot as `old_data` and the post-mutation entity as `new_data`, so update/delete handlers must load the old row before writing.

### Routing structure (in `server.go`)

Routes are grouped under `/api/v1` (behind the per-IP rate limiter) with three tiers:
1. **Public** — no auth: artist/label/source listing, organisation listing (`/organisation/all` plus the type-specific `/organisation/labels`, `/organisation/distributors`, `/organisation/cults`), login, suggestion/feedback submission.
2. **Protected** (`AuthMiddleware`) — any valid JWT: logout, me, artist admin CRUD + stats, label CRUD, suggestion admin review, feedback admin review, and change history (`/history`, `/history/editors`).
3. **Admin-only** (`AuthMiddleware` + `RequireRole("ADMIN")`) — user management under `/auth`, source CRUD, and organisation create/update/delete.

Outside `/api/v1`: `GET /health` returns 204, and the `logger` middleware skips `/`, `/health`, and `/metrics`. A `requestTimeout(30 * time.Second)` middleware caps request duration.

`router.SetTrustedProxies([]string{"172.19.0.0/16"})` trusts only the Docker network Traefik sits on (in production the API is fronted by Traefik on the `traefik-public` network), so `X-Forwarded-For` yields a real `ClientIP()` for rate limiting without letting arbitrary clients spoof it.

## Database

- **artists** — core entity with Spotify metadata, UA/EN descriptions, free-text `notes`, and a `total_priority` column auto-recalculated by a trigger on `artist_labels` insert/update. Indexed on `(total_priority DESC, updated_at DESC, id ASC)` and `(updated_at DESC, id ASC)` for the default/secondary admin sort orders, plus trigram indexes for single-term substring search and a hash index on `spotify_id` (`00007`), and `lower(name)` for multi-term exact search (`00012`).
- **labels** — reference table; carries a `priority` weight (label names: `approved`, `blocked`, `warning`, `unknown`, `pride`, `base`).
- **artist_labels** — many-to-many join between artists and labels.
- **artist_countries** — per-artist country codes (`code`, ordered by `position`), each optionally linked to `evidence_sources` rows via `source_id` — since `00010` a country can carry multiple sources. There is no standalone `countries` reference table — countries are plain ISO codes, not rows with names.
- **evidence_sources** — catalog of how a country was determined for an artist, bilingual via `name` (Ukrainian) + `name_en`; the short-lived `name_uk` column added in `00005` was dropped again in `00009`. Full CRUD via `/source`.
- **organisations** — labels/distributors/cults (`type`), with `origin`, UA/EN descriptions, `notes`, and a `recommendation` constrained to a fixed set of Ukrainian phrases. Indexed on `(type)` and `(name)`.
- **change_history** — append-only audit log (`entity_type`, `entity_id`, `entity_name`, `action`, `editor_username`, `old_data`/`new_data` JSONB), with per-filter indexes that embed the `(created_at DESC, id DESC)` ordering used by `/history`.
- **suggestions**, **feedbacks** — public-submission tables for new-artist suggestions and user feedback, with admin-only review/listing endpoints.

Base schema is `internal/migrations/00001_create_tables.sql`; `00002_admin_setup.sql` adds admin user setup and the `users`/`suggestions`/`feedbacks`/`evidence_sources`/`artist_countries` tables. Later migrations (`00003`–`00012`) cover source translations, sort/search indexes, organisations, multi-source countries, change history, and the `lower(name)` search index. New migrations follow the `NNNNN_description.sql` naming convention, use tern's `---- create above / drop below ----` separator, and are picked up automatically via `embed.go`. **CI fails a change that adds more than one migration file** — ship at most one migration per deploy so rollback stays safe.

## Containers

- `Dockerfile.dev` — Alpine + `air` for hot reload; source bind-mounted.
- `Dockerfile.prod` — multi-stage: `golang:1.26.2-alpine` builder (CGO-disabled static binary) → `alpine:latest` runtime, runs as non-root `app` user.
- `docker-compose.dev.yaml` — local API + Postgres 17 with healthchecks; dev `JWT_SECRET`/`DB_URL` are inline, so no `.env` is needed to boot locally.
- `docker-compose.prod.yaml` — production compose, deployed via the `CD` GitHub Actions workflow over SSH. Serves `api.phonkersbase.com` through Traefik (labels on the `traefik-public` network, TLS via `myresolver`). Passes through `OTEL_EXPORTER_OTLP_ENDPOINT`/`_PROTOCOL`/`_HEADERS` from the server's `.env` (not in this repo — set manually alongside `JWT_SECRET`).

## CI/CD

- **CI** (`.github/workflows/ci.yaml`): on push/PR to `main` — `go mod download`, `go build ./...`, `go test ./...`, `go vet ./...`, plus the single-migration-per-change guard.
- **CD** (`.github/workflows/cd.yaml`): triggers after CI succeeds on `main`. Detects whether API code (`internal/`, `cmd/`, `go.mod`, `go.sum`, `Dockerfile.prod`) or `docker-compose.prod.yaml` changed (by diffing against the currently deployed image's revision label) and only rebuilds/redeploys when needed. Images tagged with `latest` and the commit SHA; deploy pulls and restarts via `docker compose` over SSH.
