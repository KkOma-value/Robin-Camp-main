# movies-api Specification

## Purpose
TBD - created by archiving change add-movie-rating-api. Update Purpose after archive.
## Requirements
### Requirement: Movies API Deployment

The system SHALL run locally via Docker Compose with one Go application container and one MySQL container, automatically applying SQL migrations and exposing `/healthz` for readiness; all configuration MUST come from environment variables (`PORT`, `AUTH_TOKEN`, `DB_URL`, `BOXOFFICE_URL`, `BOXOFFICE_API_KEY`).

#### Scenario: Local stack boots successfully

- **GIVEN** the developer runs `make docker-up` with valid environment variables
- **WHEN** Docker Compose starts both containers
- **THEN** migrations execute before the app begins accepting traffic AND `GET /healthz` returns 200.

#### Scenario: Missing configuration blocks startup

- **GIVEN** `BOXOFFICE_API_KEY` is unset
- **WHEN** the server process loads configuration
- **THEN** it SHALL fail fast with a descriptive error instead of booting with default/fallback values.

### Requirement: Movie Creation with Box Office Enrichment

`POST /movies` MUST validate inputs per `openapi.yml`, persist the movie, call `GET /boxoffice?title=` using the Apifox mock, merge successful responses into the stored record, and return `201 Created` with a `Location` header; upstream failures SHALL log warnings and leave `boxOffice=null` without blocking creation.

#### Scenario: Creation succeeds with enrichment

- **GIVEN** a new movie payload and the box office API returns 200 with revenue data
- **WHEN** the client calls `POST /movies`
- **THEN** the response status is 201 with `Location: /movies/{title}` AND the stored movie contains the `boxOffice` object from the upstream response.

#### Scenario: Upstream failure falls back to null boxOffice

- **GIVEN** the box office API returns 404 or times out
- **WHEN** the client calls `POST /movies`
- **THEN** the response status is still 201 AND the persisted movie has `boxOffice=null` while the error is logged for observability.

### Requirement: Movie Discovery & Pagination

`GET /movies` SHALL support optional filters (`q`, `year`, `genre`), respect a `limit` up to the contract maximum, and implement cursor-based pagination that returns `{items, nextCursor}` exactly as defined in `openapi.yml`.

#### Scenario: Filtered search

- **GIVEN** movies exist across multiple years and genres
- **WHEN** the client calls `GET /movies?year=2010&genre=Sci-Fi`
- **THEN** only movies released in 2010 with genre `Sci-Fi` appear in `items` AND unrelated movies are excluded.

#### Scenario: Cursor pagination

- **GIVEN** more movies exist than the requested `limit`
- **WHEN** the client calls `GET /movies?limit=1`
- **THEN** the response contains exactly one item plus a non-null `nextCursor` AND a follow-up call with that cursor returns the next page.

### Requirement: Movie Rating Submission

`POST /movies/{title}/ratings` MUST authenticate via Bearer token and `X-Rater-Id`, accept ratings only in 0.5 increments between 0.5 and 5.0, upsert on `(movie, rater)` uniqueness, and return 201 for new ratings or 200 for updates; invalid ratings SHALL return the 422 error model.

#### Scenario: New rating created

- **GIVEN** an authenticated request with `X-Rater-Id: user123` and body `{ "rating": 4.5 }`
- **WHEN** the user has not rated the movie before
- **THEN** the response is 201 with the stored rating payload echoing the user ID and value.

#### Scenario: Rating upsert updates existing value

- **GIVEN** `user123` previously rated the movie 4.5
- **WHEN** they submit `{ "rating": 3.5 }`
- **THEN** the API returns 200 and the stored rating now equals 3.5.

#### Scenario: Invalid rating rejected

- **GIVEN** a request body `{ "rating": 6 }`
- **WHEN** the endpoint processes the payload
- **THEN** it responds with 422 and the error schema defined in `openapi.yml` explaining the allowed range/step.

### Requirement: Movie Rating Aggregation

`GET /movies/{title}/rating` SHALL compute `{average, count}` from persisted ratings, rounding the average to one decimal place using standard rounding and returning 404 if the movie does not exist.

#### Scenario: Aggregation returns rounded value

- **GIVEN** two ratings 3.5 and 4.0 for the same movie
- **WHEN** the client calls `GET /movies/{title}/rating`
- **THEN** the response body is `{ "average": 3.8, "count": 2 }`.

#### Scenario: Unknown movie yields 404

- **GIVEN** a title that is not stored
- **WHEN** the client calls `GET /movies/Unknown/rating`
- **THEN** the API responds with 404 using the contract’s error body.

