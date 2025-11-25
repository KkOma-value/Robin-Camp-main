-- +goose Up
CREATE TABLE movies (
    id CHAR(26) PRIMARY KEY,
    title VARCHAR(255) NOT NULL UNIQUE,
    release_date DATE NOT NULL,
    genre VARCHAR(64) NOT NULL,
    distributor VARCHAR(255),
    budget BIGINT,
    mpa_rating VARCHAR(16),
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    INDEX idx_title (title),
    INDEX idx_release_date (release_date),
    INDEX idx_genre (genre),
    INDEX idx_distributor (distributor),
    INDEX idx_budget (budget),
    INDEX idx_mpa_rating (mpa_rating)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

