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
| `PORT` | No | `8080` | HTTP listen port |

## API Reference

All routes are prefixed with `/api/v1` and rate-limited to **50 requests/minute per IP**.

### GET `/api/v1/artist/all`

Returns a paginated list of artists.

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `locale` | `uk` \| `en` | **Required.** Language for description field |
| `search` | string | Free-text search on name and description |
| `country` | string | Comma-separated country names to filter by |
| `label` | string | Comma-separated label names to filter by |
| `offset` | int | Pagination offset (default: `0`) |
| `limit` | int | Page size, 1–200 (default: `50`) |
| `by_artist` | `asc` \| `desc` | Sort by artist name |
| `by_country` | `asc` \| `desc` | Sort by first country |
| `by_listen` | `asc` \| `desc` | Sort by total label priority |

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
      "countries": [{ "id": "1", "name": "Ukraine", "originalName": "Україна", "createdAt": "...", "updatedAt": "..." }],
      "listenLabels": [{ "id": "1", "name": "pride", "originalName": "Наша гордість 💪", "priority": 100, "createdAt": "...", "updatedAt": "..." }],
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

### GET `/api/v1/country/all`

Returns all countries ordered by name.

### GET `/api/v1/label/all`

Returns all labels ordered by priority (descending), then name.

**Label names:** `approved`, `blocked`, `warning`, `unknown`, `pride`, `base`

### POST `/api/v1/app/dev/force-sync`

Development endpoint stub. Returns `{"success": true, "message": "sync triggered"}`.

## Database Schema

- **artists** — core entity with optional Spotify metadata, description in UA/EN, country array, and computed `total_priority`
- **countries** — reference table
- **labels** — categorization labels with priority weights
- **artist_labels** — many-to-many join; a trigger automatically recalculates `total_priority` on insert/update
