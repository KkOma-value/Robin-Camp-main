# Project Context

## Purpose

Build a self-contained Movie Rating API that strictly follows `openapi.yml`, persists movies and user ratings locally, enriches new movies with box-office data from the Apifox mock, and runs end-to-end through Docker Compose without any cloud dependencies.

## Tech Stack

- Go 1.22 with `chi` router + standard library for HTTP handlers, validation, and background jobs
- PostgreSQL 15 as the primary data store; schema managed via SQL migrations executed by `goose` during container startup
- Docker/Docker Compose for local orchestration plus multi-stage Dockerfile targeting a non-root distroless runtime
- Bash + `curl` + `jq` for automation and the provided `e2e-test.sh` smoke suite

## Project Conventions

### Code Style

- Run `gofmt` + `goimports` on every commit; prefer explicit structs and constructor helpers over global vars
- Use idiomatic Go naming (mixedCaps, no underscores) and keep handler/request/response DTOs in `internal/api`
- Enforce linting with `golangci-lint` (default config) before opening a PR; no unused exports
- Return typed errors (`errors.Join`/`fmt.Errorf`) and wrap external failures with context so logs stay actionable

### Architecture Patterns

- Layered/hexagonal layout: `cmd/` wiring, `internal/api` for transport, `internal/core` for business rules, `internal/store` for persistence, `internal/clients` for outbound HTTP (Box Office)
- Dependency injection via lightweight constructors; avoid global singletons to keep tests isolated
- PostgreSQL access through the standard `database/sql` driver + `pgx` pool and repository interfaces for easier fakes
- Outbound Box Office integration wrapped in a client that handles retries, timeouts, and graceful degradation

### Testing Strategy

- Table-driven unit tests for handlers, services, and validation logic using Go’s `testing` package + `testify/require`
- Integration tests that run against a disposable Postgres container (via `docker compose` or `testcontainers-go`) to cover migrations and repositories
- Contract-level verification through the provided `e2e-test.sh`, wired into `make test-e2e` so CI replicates reviewer flow
- Future enhancement: add Spectral or `openapi-diff` checks to ensure responses never drift from `openapi.yml`

### Git Workflow

- Trunk-based flow: feature branches off `main`, short-lived (<1 day) with required PR review before merge
- Conventional Commits (`feat:`, `fix:`, `chore:`) to keep change history searchable and enable automated release notes later
- Rebase onto latest `main` before merging to avoid merge commits; force-push allowed only on personal branches
- Tag milestones after the e2e suite passes (`v0.x.y`) so reviewers can reproduce artifacts easily

## Domain Context

- Movies are uniquely identified by title per spec; creation must fetch optional box-office metadata via `GET /boxoffice?title=` yet remain resilient when the mock fails (store `boxOffice=null`)
- Ratings are keyed by `(movie_title, rater_id)` and must support upsert semantics with half-point increments between 0.5 and 5.0 inclusive; aggregated averages are rounded to one decimal place
- `GET /movies` supports compound filters (`q`, `year`, `genre`) plus cursor pagination returning `{items,nextCursor}`; responses must mirror `openapi.yml` schema including `Location` headers on `201`
- Auth is static: Bearer token for write endpoints, `X-Rater-Id` required for rating submissions; everything configurable through environment variables only

## Important Constraints

- Must operate fully offline/local—no managed cloud services, only Dockerized dependencies
- API behavior, status codes, and payloads cannot deviate from `openapi.yml`; any contract change requires an approved OpenSpec proposal
- Configuration (DB URL, tokens, upstream URLs) must come from env vars; secrets cannot be baked into code or images
- Containers must expose health checks (`/healthz` for app, Compose healthcheck for Postgres) before tests run

## External Dependencies

- Apifox Box Office mock (`BOXOFFICE_URL`, `BOXOFFICE_API_KEY`) for revenue enrichment; called via HTTPS with tight timeouts/retries
- Official `postgres:15-alpine` image (or equivalent) managed by Docker Compose with initialized migrations
- Local tooling: `docker`, `docker compose`, `make`, `bash`, `curl`, `jq`, and `golangci-lint`
