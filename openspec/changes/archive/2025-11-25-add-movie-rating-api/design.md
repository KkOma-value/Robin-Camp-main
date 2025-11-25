# Design

## Context

- No service currently implements the `openapi.yml` contract or persists movies/ratings.
- The assignment requires a local-only deployment (Docker Compose + Makefile) with configuration provided exclusively through environment variables.
- Every movie creation must consult the Apifox Box Office mock; failures cannot block persistence.

## Goals

- Deliver a Go 1.22 HTTP service that satisfies every endpoint, auth check, and error model defined in `openapi.yml`.
- Persist movies, ratings, and optional box-office data in MySQL with migrations that auto-run during container startup.
- Provide deterministic local tooling (Dockerfile, docker-compose.yml, Make targets, `.env.example`, `e2e-test.sh` integration).
- Document architecture and schema decisions so implementation stays aligned with the spec.

## Non-Goals

- Deploying to cloud platforms or introducing additional services beyond the API + database.
- Building a UI or any extra endpoints not described in the OpenAPI file.
- Integrating real-world box office providers beyond the supplied mock.

## Architecture Overview

- **Process**: single Go binary (`cmd/server`) listening on `0.0.0.0:8080`, orchestrated via Docker Compose with a companion MySQL container.
- **Routing & middleware**: `chi` handles HTTP routing plus middleware for auth (`Authorization` bearer token + `X-Rater-Id`), logging, pagination, and validation.
- **Layering**: `internal/api` (transport) → `internal/core` (use cases) → `internal/store` (repositories using `database/sql` + `sqlx`) → MySQL.
- **External client**: `internal/clients/boxoffice` wraps `net/http` with 2s timeout, two retries on 5xx/timeouts, and marks failures as non-blocking.
- **Config & lifecycle**: config struct populated from env vars, dependency-injected into modules; graceful shutdown via context cancellation on SIGTERM/SIGINT.
- **Observability**: structured logs via `log/slog`, request logging middleware, and `/healthz` for liveness checks.

## Data Model (MySQL)

All timestamps are UTC (`TIMESTAMP(6)`), IDs are ULIDs encoded as `CHAR(26)`.

### `movies`

- `id CHAR(26) PRIMARY KEY`
- `title VARCHAR(255) NOT NULL UNIQUE`
- `release_date DATE NOT NULL`
- `genre VARCHAR(64) NOT NULL`
- `distributor VARCHAR(255)`
- `budget BIGINT` (stored as cents to avoid floats)
- `mpa_rating VARCHAR(16)`
- `created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)`
- `updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)`

### `movie_box_office`

- `movie_id CHAR(26) PRIMARY KEY REFERENCES movies(id) ON DELETE CASCADE`
- `gross_usd BIGINT NOT NULL`
- `currency VARCHAR(8) NOT NULL`
- `source VARCHAR(64) NOT NULL`
- `last_reported TIMESTAMP(6) NOT NULL`
- `fetched_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)`

### `movie_ratings`

- `movie_id CHAR(26) NOT NULL REFERENCES movies(id) ON DELETE CASCADE`
- `rater_id VARCHAR(128) NOT NULL`
- `rating DECIMAL(2,1) NOT NULL CHECK (rating IN (0.5,1.0,1.5,2.0,2.5,3.0,3.5,4.0,4.5,5.0))`
- `updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)`
- `PRIMARY KEY (movie_id, rater_id)` to enforce upsert semantics

Rating aggregates use `SELECT ROUND(AVG(rating), 1) AS average, COUNT(*) FROM movie_ratings WHERE movie_id = ?`. Cursor pagination encodes the last `(created_at, id)` pair as base64 JSON for deterministic `GET /movies` traversal.

## Key Decisions

- **Go + chi** for a minimal, idiomatic HTTP stack.
- **MySQL 8** to respect the user's request; use `goose` for SQL migrations executed before the app starts.
- **ULIDs** to provide sortable, globally unique IDs usable both in DB and API responses.
- **Config validation** at startup to fail fast when required environment variables are missing.
- **Fail-open box office enrichment** so upstream outages never block writes.

## Risks & Mitigations

- **Box Office instability** → Bound retries/timeouts, log warnings, and persist movie sans box office payload.
- **DB readiness race** → Compose healthcheck waits for MySQL; application also retries initial connections with backoff.
- **Cursor tampering** → Validate/decode cursor payloads, return 400 on malformed input to avoid inconsistent pagination.
- **Secrets in code** → Enforce `.env` + runtime config only; forbid hard-coded tokens/URLs.

## Migration Plan

1. Merge this proposal/spec change.
2. Scaffold Go module, configs, Docker assets, and Make targets.
3. Create MySQL migrations plus entrypoint that runs `goose up` automatically.
4. Implement repositories, services, handlers, and box office client iteratively.
5. Add auth middleware, validation, pagination, and `/healthz`.
6. Run unit tests, `golangci-lint`, and `make test-e2e` before delivery.

## Open Questions

- None. Requirements are fully defined by `ASSIGNMENT.md` and `openapi.yml`.
