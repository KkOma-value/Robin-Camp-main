# Tasks

## 1. Scaffolding & Environment

- [x] 1.1 Initialize Go module (`go mod init`), add dependencies (`chi`, `sqlx`, `ulid`, `slog`, `golangci-lint` config).
- [x] 1.2 Define configuration loader validating `PORT`, `AUTH_TOKEN`, `DB_URL`, `BOXOFFICE_URL`, `BOXOFFICE_API_KEY`.
- [x] 1.3 Create shared logger + HTTP server bootstrap wiring graceful shutdown.

## 2. Database Layer

- [x] 2.1 Author `goose` SQL migrations for `movies`, `movie_box_office`, and `movie_ratings` tables (per design).
- [x] 2.2 Implement MySQL connection factory with retry/backoff and health pings.
- [x] 2.3 Build repository interfaces + implementations for movies, ratings, pagination, and aggregates.

## 3. External Integrations

- [x] 3.1 Implement Box Office client with env-driven base URL/API key, 2s timeout, retries, and structured error handling.
- [x] 3.2 Add feature toggle/error path so movie creation succeeds when the mock returns non-200 or times out.

## 4. HTTP/API Layer

- [x] 4.1 Implement middleware for auth (Bearer token) and `X-Rater-Id` enforcement, plus request/response logging.
- [x] 4.2 Build `/healthz` handler returning 200 when DB reachable.
- [x] 4.3 Implement `POST /movies` with validation, DB persistence, box-office enrichment, and `Location` header.
- [x] 4.4 Implement `GET /movies` with filters (`q`, `year`, `genre`), cursor pagination, and schema-compliant response.
- [x] 4.5 Implement `POST /movies/{title}/ratings` with rating validation (0.5 steps), upsert semantics, and 201/200 responses.
- [x] 4.6 Implement `GET /movies/{title}/rating` returning `{average,count}` with rounding to one decimal place.
- [x] 4.7 Standardize error handling to mirror `openapi.yml` (400/401/403/404/422 models).

## 5. Tooling & Ops

- [x] 5.1 Write Dockerfile (multi-stage, non-root runtime) and docker-compose.yml including MySQL + healthchecks + migration hook.
- [x] 5.2 Add Make targets (`docker-up`, `docker-down`, `test-e2e`) and ensure `.env.example` lists every required variable.
- [x] 5.3 Integrate `golangci-lint` and unit test targets; ensure CI/docs instruct manual README authoring per assignment.
- [ ] 5.4 Run unit tests, `golangci-lint`, and `make test-e2e`; fix any failures before submission.
