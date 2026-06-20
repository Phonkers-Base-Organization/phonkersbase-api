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
  middlewares/ratelimit.go   # Per-IP rate limiting
  migrations/                # Embedded SQL migrations
  repository/                # PostgreSQL data access
  server/server.go           # Router setup & server startup
```

## Development

**Run with hot-reload:**
```bash
docker compose up
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_URL` | Yes | — | PostgreSQL connection string |
| `JWT_SECRET` | Yes | — | JWT signing secret, min 32 chars |
| `CORS_ORIGIN` | No | `http://localhost:3000` | Comma-separated list of allowed CORS origins |
| `PORT` | No | `8080` | HTTP listen port |

## API Reference

All routes are prefixed with `/api/v1` and rate-limited to **50 requests/minute per IP**.

### GET `/api/v1/artist/all`

Returns a paginated list of artists. Public endpoint.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `locale` | `uk` \| `en` | Language for description field (default: `uk`) |
| `search` | string | Free-text search on name, description, and Spotify ID |
| `country` | string | Comma-separated ISO country codes to filter by (e.g. `UA,PL`) |
| `label` | string | Comma-separated label names to filter by |
| `offset` | int | Pagination offset (default: `0`) |
| `limit` | int | Page size, 1–200 (default: `50`) |
| `by_artist` | `asc` \| `desc` | Sort by artist name |
| `by_country` | `asc` \| `desc` | Sort by first country code |
| `by_listen` | `asc` \| `desc` | Sort by total label priority |

If none of the sort params are given, results default to `total_priority DESC, updated_at DESC`. Any combination of sort params can be combined; `id ASC` is always appended as a final tiebreaker.

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
          "source": { "id": "1", "name": "Spotify опис", "nameUk": "Spotify опис", "nameEn": "Spotify description" }
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

### Admin-only endpoints (require `ADMIN` role)

- `POST`/`PUT`/`DELETE` `/api/v1/label`, `/api/v1/label/:id` — label CRUD (protected, any authenticated user)
- `POST`/`PUT`/`DELETE` `/api/v1/source`, `/api/v1/source/:id` — evidence source CRUD (admin-only)
- `GET`/`POST`/`PUT`/`DELETE` `/api/v1/artist/admin/all`, `/api/v1/artist`, `/api/v1/artist/:id` — artist CRUD and admin listing (protected)
- `GET` `/api/v1/artist/stats` — artist counts/last-added stats (protected)
- `GET`/`PATCH` `/api/v1/suggestion`, `/api/v1/suggestion/:id/status` — review submitted artist suggestions (protected)
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
