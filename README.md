# PhonkersBase API v2

REST API for PhonkersBase. Built with Go, Gin, and PostgreSQL.

## Stack

- **Go 1.26** · **Gin** web framework
- **PostgreSQL 17** via `pgx/v5` with connection pooling
- **Tern** for database migrations
- **Zerolog** for structured logging
- **Air** for hot-reload in development

## Project Structure

```
cmd/api/main.go              # Entry point
internal/
  config/config.go           # Env variable loading & validation
  domain/models.go           # Domain types
  handlers/                  # HTTP handlers
  metrics/                   # OpenTelemetry metrics (HTTP/DB latency, errors, Go runtime)
  middlewares/ratelimit.go   # Per-IP rate limiting
  migrations/                # Embedded SQL migrations
  repository/                # PostgreSQL data access
  server/server.go           # Router setup & server startup
```

## Development

**Run with hot-reload:**
```bash
docker compose -f docker-compose.dev.yaml up
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_URL` | Yes | — | PostgreSQL connection string |
| `JWT_SECRET` | Yes | — | JWT signing secret, min 32 chars |
| `CORS_ORIGIN` | No | `http://localhost:3000` | Comma-separated list of allowed CORS origins |
| `PORT` | No | `8080` | HTTP listen port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | — | OTLP endpoint for metrics export. Unset disables metrics (instruments become no-ops) |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | No | — | OTLP protocol, e.g. `http/protobuf` |
| `OTEL_EXPORTER_OTLP_HEADERS` | No | — | OTLP request headers, e.g. `Authorization=Basic <base64>` for Grafana Cloud |
| `LOKI_URL` | No | — | Required for Loki docker driver |

## Observability

- `GET /health` — liveness check (204), skipped by request logging.
- Metrics are collected via OpenTelemetry and pushed over OTLP (no `/metrics` scrape endpoint — see `internal/metrics/`):
  - `http.server.request.duration` — HTTP latency, labeled by method/route/status
  - `db.client.query.duration` — Postgres query latency, labeled by SQL verb and error
  - `app.errors` — counter incremented automatically on every `Error`/`Fatal` zerolog event
  - Go runtime metrics (GC, heap, goroutines) via `go.opentelemetry.io/contrib/instrumentation/runtime`
- Structured JSON logs via `zerolog`, one line per request (method, path, status, latency, IP).

## API Reference

All routes are prefixed with `/api/v1` and rate-limited to **50 requests/minute per IP**.

### GET `/api/v1/artist/all`

Returns a paginated list of artists. Public endpoint.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `locale` | `uk` \| `en` | Language for description field (default: `uk`) |
| `search` | string | One term, or a comma-separated list — see [Search semantics](#search-semantics) |
| `country` | string | Comma-separated ISO country codes to filter by (e.g. `UA,PL`) |
| `label` | string | Comma-separated label names to filter by |
| `offset` | int | Pagination offset (default: `0`) |
| `limit` | int | Page size, 1–200 (default: `50`) |
| `by_artist` | `asc` \| `desc` | Sort by artist name |
| `by_country` | `asc` \| `desc` | Sort by first country code |
| `by_listen` | `asc` \| `desc` | Sort by total label priority |

If none of the sort params are given, results default to `total_priority DESC, updated_at DESC`. Any combination of sort params can be combined; `id ASC` is always appended as a final tiebreaker.

#### Search semantics

`search` is split on commas into terms (surrounding whitespace and empty terms are ignored), and
**the number of terms decides how they are matched**:

| Terms | Matching |
|-------|----------|
| exactly 1 | **Substring**, case-insensitive, against artist name, the localized description (`description_ua` or `description_en` per `locale`) and the Spotify link — for a search box where the user types a fragment |
| 2 or more | **Exact**, case-insensitive, against artist name |

In both modes a term that is a bare 22-character Spotify ID, an `open.spotify.com/artist/<id>` URL, or
a `spotify:artist:<id>` URI is resolved to that artist's `spotify_id` rather than being matched as text.
Terms are OR'd together, so the result is every artist matching any term.

Multi-term search exists for bulk existence checks — a caller enumerating a tracklist already knows the
exact names or IDs it is asking about, which is why exact matching is the right semantic there. It is also
the only way for it to be fast: substring-matching 46 terms expands to 139 `ILIKE` predicates that Postgres
resolves with a sequential scan (~500 ms and rising linearly with term count and table size), while exact
matching is a single index lookup regardless of term count (~1.5 ms). It is more accurate, too: a short
term like `Muse` substring-matches unrelated names such as `MYSTICMUSE`.

Two consequences worth knowing:

- **A comma always separates terms.** An artist whose name contains a comma cannot be searched as a single
  multi-term entry, and a collaboration credit like `Artist A, Artist B` is treated as two separate lookups.
- **Encode each term once.** Double-encoding (sending `%252C` for a comma inside a term) leaves a literal
  `%2C` in the term, which will not match anything.

Prefer one request with many terms over many single-term requests — the rate limit is per IP, and a
50-term lookup costs roughly the same as a 1-term one.

**Response**

```json
{
  "items": [
    {
      "id": "1",
      "name": "PRXZY",
      "link": "https://open.spotify.com/...",
      "spotifyId": "...",
      "avatarUrl": "https://...",
      "description": "...",
      "descriptionEn": "...",
      "notes": "...",
      "countries": [
        {
          "code": "UA",
          "sources": [{ "id": "1", "name": "Spotify опис", "nameEn": "Spotify description" }]
        }
      ],
      "listenLabels": [{ "id": "1", "name": "pride" }],
      "createdAt": "...",
      "updatedAt": "..."
    }
  ],
  "info": {
    "limit": 50,
    "offset": 0,
    "total": 50,
    "totalPages": 1,
    "currentPage": 1
  }
}
```

### GET `/api/v1/label/all`

Returns all labels ordered by priority (descending), then name. Public endpoint.

**Label names:** `approved`, `blocked`, `warning`, `unknown`, `pride`, `base`

### GET `/api/v1/source/all`

Returns all evidence sources (used to record how an artist's country was determined), with bilingual (`nameUk`/`nameEn`) names. Public endpoint.

### GET `/api/v1/organisation/all`

Returns a paginated list of organisations (labels, distributors and cults tracked alongside artists),
ordered by name. Public endpoint.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | `Label` \| `Distributor` \| `Cult` | Filter by organisation type. Not validated — an unrecognized value simply matches nothing |
| `search` | string | Case-insensitive substring match on the organisation name |
| `offset` | int | Pagination offset (default: `0`) |
| `limit` | int | Page size, 1–500 (default: `50`); out-of-range values return `400` |

**Response**

```json
{
  "items": [
    {
      "id": "1",
      "name": "Example Records",
      "link": "https://...",
      "origin": "RU",
      "description": "...",
      "descriptionEn": "...",
      "notes": "...",
      "type": "Label",
      "recommendation": "Не використовуй",
      "createdAt": "...",
      "updatedAt": "..."
    }
  ],
  "info": { "limit": 50, "offset": 0, "total": 1, "totalPages": 1, "currentPage": 1 }
}
```

Note that the Ukrainian description is serialized as `description` (not `descriptionUk`), matching the
artist payload. `recommendation` is one of five fixed Ukrainian phrases, enforced by both a check
constraint and the API: `Не використовуй`, `Не слухай це`, `Будь обережний`, `Можеш використовувати`,
`Можеш слухати`.

### GET `/api/v1/organisation/labels`, `/api/v1/organisation/distributors`, `/api/v1/organisation/cults`

Type-specific shorthands for `/organisation/all?type=Label|Distributor|Cult`. Same `search`, `offset`
and `limit` parameters, same response shape. Public endpoints.

### Organisation CRUD (admin-only)

`POST /api/v1/organisation`, `PUT /api/v1/organisation/:id`, `DELETE /api/v1/organisation/:id` — require
the `ADMIN` role. `POST` returns `201` with the created organisation, `PUT` returns `200` with the updated
one, `DELETE` returns `204`; a missing `:id` returns `404`.

**Request body** (`POST` and `PUT`)

```json
{
  "name": "Example Records",
  "link": "https://...",
  "origin": "RU",
  "description": "...",
  "descriptionEn": "...",
  "notes": "...",
  "type": "Label",
  "recommendation": "Не використовуй"
}
```

`name`, `origin`, `type` and `recommendation` are required; an unrecognized `recommendation` returns `400`.
All three mutations are recorded in the change history.

### GET `/api/v1/history`

Returns a paginated change log of admin mutations, newest first (`created_at DESC, id DESC`).
Requires authentication (any valid JWT).

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `entityType` | `artist` \| `label` \| `source` \| `organisation` | Filter by entity type; any other value returns `400` |
| `entityId` | string | Filter to a single entity's history |
| `action` | `create` \| `update` \| `delete` | Filter by action; any other value returns `400` |
| `editor` | string | Filter by editor username (exact) |
| `search` | string | Case-insensitive substring match on the entity name recorded at the time of the change |
| `offset` | int | Pagination offset (default: `0`) |
| `limit` | int | Page size, 1–500 (default: `50`); out-of-range values return `400` |

**Response**

```json
{
  "items": [
    {
      "id": "1",
      "entityType": "artist",
      "entityId": "3414",
      "entityName": "KAITO SHOMA",
      "action": "update",
      "editorId": "1",
      "editorUsername": "admin",
      "oldData": { "...": "entity snapshot before the change, null on create" },
      "newData": { "...": "entity snapshot after the change, null on delete" },
      "createdAt": "..."
    }
  ],
  "info": { "limit": 50, "offset": 0, "total": 1, "totalPages": 1, "currentPage": 1 }
}
```

Entries are written best-effort by the mutating handlers: a failed history insert is logged but never
fails the mutation itself, so the log is an audit aid rather than a guaranteed-complete record.
`entityName` is the name as it stood at the time of the change, which is why it is stored rather than
joined — the entity may since have been renamed or deleted.

### GET `/api/v1/history/editors`

Returns the distinct editor usernames present in the change log, for populating a filter dropdown.
Requires authentication (any valid JWT).

```json
{ "items": ["admin", "editor1"] }
```

### Admin-only endpoints (require `ADMIN` role)

- `POST`/`PUT`/`DELETE` `/api/v1/label`, `/api/v1/label/:id` — label CRUD (protected, any authenticated user)
- `POST`/`PUT`/`DELETE` `/api/v1/source`, `/api/v1/source/:id` — evidence source CRUD (admin-only)
- `GET`/`POST`/`PUT`/`DELETE` `/api/v1/artist/admin/all`, `/api/v1/artist`, `/api/v1/artist/:id` — artist CRUD and admin listing (protected)
- `GET` `/api/v1/artist/stats` — artist counts/last-added stats (protected)
- `GET`/`DELETE` `/api/v1/suggestion`, `/api/v1/suggestion/:id` — review submitted artist suggestions (protected)
- `GET`/`DELETE` `/api/v1/feedback`, `/api/v1/feedback/:id` — review submitted feedback (protected)
- `/api/v1/auth/*` — login (public), logout/me (protected), user management (admin-only)

## Database Schema

- **artists** — core entity with optional Spotify metadata, description in UA/EN, free-text `notes`, and computed `total_priority`
- **labels** — categorization labels with priority weights
- **artist_labels** — many-to-many join; a trigger automatically recalculates `total_priority` on insert/update
- **artist_countries** — per-artist ISO country codes (ordered by `position`), each optionally linked to an `evidence_sources` row
- **evidence_sources** — bilingual catalog of how a country was determined for an artist
- **suggestions**, **feedbacks** — public submissions awaiting admin review
- **users** — admin/editor accounts for JWT auth
