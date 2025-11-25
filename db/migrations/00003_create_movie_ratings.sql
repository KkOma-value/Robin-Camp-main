-- +goose Up
CREATE TABLE movie_ratings (
    movie_id CHAR(26) NOT NULL,
    rater_id VARCHAR(128) NOT NULL,
    rating DECIMAL(2,1) NOT NULL,
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (movie_id, rater_id),
    CONSTRAINT fk_movie_ratings_movie
        FOREIGN KEY (movie_id) REFERENCES movies(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_rating_values
        CHECK (rating IN (0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;