-- +goose Up
CREATE TABLE movie_box_office (
    movie_id CHAR(26) PRIMARY KEY,
    gross_usd BIGINT NOT NULL,
    opening_weekend_usa BIGINT,
    currency VARCHAR(8) NOT NULL,
    source VARCHAR(64) NOT NULL,
    last_reported TIMESTAMP(6) NOT NULL,
    fetched_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_movie_box_office_movie
        FOREIGN KEY (movie_id) REFERENCES movies(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
