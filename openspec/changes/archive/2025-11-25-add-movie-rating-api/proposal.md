# Change: Add Movie Rating API

## Why

- The take-home requires a fully compliant Movie Rating API that matches `openapi.yml`, including MySQL persistence, box office enrichment, and Docker-based local execution.
- No current specs or implementation exist, so we need to define behavior, architecture, and database design up front to keep future work aligned and testable.

## What Changes

- Introduce a Go 1.22 service exposing every contract path in `openapi.yml`, including `/movies`, `/movies/{title}/ratings`, `/movies/{title}/rating`, and `/healthz`.
- Persist movies, box-office metadata, and user ratings in MySQL with migrations and an auto-run bootstrap step wired into Docker Compose.
- Call the Apifox Box Office mock during movie creation with resilient retries/timeouts; merge successful responses into stored records while tolerating failures.
- Add Make targets (`docker-up`, `docker-down`, `test-e2e`) plus `.env.example` entries so reviewers can run the stack and the provided `e2e-test.sh`.
- Provide observability/ops basics: structured logging, health checks, graceful shutdown, and configuration solely via environment variables.

## Impact

- Affected specs: `movies-api` (new capability describing endpoints, data rules, and integration responsibilities).
- Affected code: new Go modules under `cmd/server`, `internal/api`, `internal/core`, `internal/store`, `internal/clients/boxoffice`, migrations under `db/migrations`, Dockerfile, docker-compose, Makefile, `.env.example`, README, and CI helpers.
- Tooling: introduces dependencies on `golang`, `mysql` container, `migrate`/`goose`, and `golangci-lint`.
