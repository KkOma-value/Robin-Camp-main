package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Movie represents a movie record.
type Movie struct {
	ID          string     `db:"id" json:"id"`
	Title       string     `db:"title" json:"title"`
	ReleaseDate time.Time  `db:"release_date" json:"releaseDate"`
	Genre       string     `db:"genre" json:"genre"`
	Distributor *string    `db:"distributor" json:"distributor,omitempty"`
	Budget      *int64     `db:"budget" json:"budget,omitempty"`
	MPARating   *string    `db:"mpa_rating" json:"mpaRating,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"-"`
	UpdatedAt   time.Time  `db:"updated_at" json:"-"`
	BoxOffice   *BoxOffice `json:"boxOffice,omitempty"`
}

// BoxOffice represents box office data.
type BoxOffice struct {
	Revenue struct {
		Worldwide         int64  `json:"worldwide"`
		OpeningWeekendUSA *int64 `json:"openingWeekendUSA,omitempty"`
	} `json:"revenue"`
	Currency    string    `json:"currency"`
	Source      string    `json:"source"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// BoxOfficeRow represents the database row for box office data.
type BoxOfficeRow struct {
	MovieID           string    `db:"movie_id"`
	GrossUSD          int64     `db:"gross_usd"`
	OpeningWeekendUSA *int64    `db:"opening_weekend_usa"`
	Currency          string    `db:"currency"`
	Source            string    `db:"source"`
	LastReported      time.Time `db:"last_reported"`
	FetchedAt         time.Time `db:"fetched_at"`
}

// Rating represents a movie rating.
type Rating struct {
	MovieID   string    `db:"movie_id"`
	RaterID   string    `db:"rater_id"`
	Rating    float64   `db:"rating"`
	UpdatedAt time.Time `db:"updated_at"`
}

// RatingAggregate represents aggregated rating statistics.
type RatingAggregate struct {
	Average float64 `db:"average" json:"average"`
	Count   int     `db:"count" json:"count"`
}

// MovieStore handles movie persistence.
type MovieStore struct {
	db *DB
}

// NewMovieStore creates a new MovieStore.
func NewMovieStore(db *DB) *MovieStore {
	return &MovieStore{db: db}
}

// Create inserts a new movie.
func (s *MovieStore) Create(ctx context.Context, movie *Movie) error {
	query := `
		INSERT INTO movies (id, title, release_date, genre, distributor, budget, mpa_rating)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		movie.ID, movie.Title, movie.ReleaseDate, movie.Genre,
		movie.Distributor, movie.Budget, movie.MPARating,
	)
	return err
}

// GetByTitle retrieves a movie by title.
func (s *MovieStore) GetByTitle(ctx context.Context, title string) (*Movie, error) {
	var movie Movie
	query := `SELECT id, title, release_date, genre, distributor, budget, mpa_rating, created_at, updated_at
	          FROM movies WHERE title = ?`
	err := s.db.GetContext(ctx, &movie, query, title)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &movie, nil
}

// GetByID retrieves a movie by ID.
func (s *MovieStore) GetByID(ctx context.Context, id string) (*Movie, error) {
	var movie Movie
	query := `SELECT id, title, release_date, genre, distributor, budget, mpa_rating, created_at, updated_at
	          FROM movies WHERE id = ?`
	err := s.db.GetContext(ctx, &movie, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &movie, nil
}

// SetBoxOffice stores box office data for a movie.
func (s *MovieStore) SetBoxOffice(ctx context.Context, movieID string, bo *BoxOfficeRow) error {
	query := `
		INSERT INTO movie_box_office (movie_id, gross_usd, opening_weekend_usa, currency, source, last_reported)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			gross_usd = VALUES(gross_usd),
			opening_weekend_usa = VALUES(opening_weekend_usa),
			currency = VALUES(currency),
			source = VALUES(source),
			last_reported = VALUES(last_reported),
			fetched_at = CURRENT_TIMESTAMP(6)
	`
	_, err := s.db.ExecContext(ctx, query,
		movieID, bo.GrossUSD, bo.OpeningWeekendUSA, bo.Currency, bo.Source, bo.LastReported,
	)
	return err
}

// GetBoxOffice retrieves box office data for a movie.
func (s *MovieStore) GetBoxOffice(ctx context.Context, movieID string) (*BoxOfficeRow, error) {
	var bo BoxOfficeRow
	query := `SELECT movie_id, gross_usd, opening_weekend_usa, currency, source, last_reported, fetched_at
	          FROM movie_box_office WHERE movie_id = ?`
	err := s.db.GetContext(ctx, &bo, query, movieID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &bo, nil
}

// Cursor represents a pagination cursor.
type Cursor struct {
	CreatedAt time.Time `json:"c"`
	ID        string    `json:"i"`
}

// EncodeCursor encodes a cursor to base64.
func EncodeCursor(c Cursor) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// DecodeCursor decodes a cursor from base64.
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	return &c, nil
}

// ListFilters represents query filters for listing movies.
type ListFilters struct {
	Query       string
	Year        *int
	Genre       string
	Distributor string
	Budget      *int64
	MPARating   string
	Limit       int
	Cursor      *Cursor
}

// List retrieves movies with filters and pagination.
func (s *MovieStore) List(ctx context.Context, filters ListFilters) ([]Movie, *Cursor, error) {
	if filters.Limit <= 0 {
		filters.Limit = 20
	}

	query := `SELECT id, title, release_date, genre, distributor, budget, mpa_rating, created_at, updated_at
	          FROM movies WHERE 1=1`
	args := []interface{}{}

	if filters.Cursor != nil {
		query += ` AND (created_at > ? OR (created_at = ? AND id > ?))`
		args = append(args, filters.Cursor.CreatedAt, filters.Cursor.CreatedAt, filters.Cursor.ID)
	}

	if filters.Query != "" {
		query += ` AND title LIKE ?`
		args = append(args, "%"+filters.Query+"%")
	}

	if filters.Year != nil {
		query += ` AND YEAR(release_date) = ?`
		args = append(args, *filters.Year)
	}

	if filters.Genre != "" {
		query += ` AND LOWER(genre) = LOWER(?)`
		args = append(args, filters.Genre)
	}

	if filters.Distributor != "" {
		query += ` AND LOWER(distributor) = LOWER(?)`
		args = append(args, filters.Distributor)
	}

	if filters.Budget != nil {
		query += ` AND budget <= ?`
		args = append(args, *filters.Budget)
	}

	if filters.MPARating != "" {
		query += ` AND mpa_rating = ?`
		args = append(args, filters.MPARating)
	}

	query += ` ORDER BY created_at, id LIMIT ?`
	args = append(args, filters.Limit+1)

	var movies []Movie
	if err := s.db.SelectContext(ctx, &movies, query, args...); err != nil {
		return nil, nil, err
	}

	var nextCursor *Cursor
	if len(movies) > filters.Limit {
		last := movies[filters.Limit-1]
		nextCursor = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		movies = movies[:filters.Limit]
	}

	return movies, nextCursor, nil
}

// RatingStore handles rating persistence.
type RatingStore struct {
	db *DB
}

// NewRatingStore creates a new RatingStore.
func NewRatingStore(db *DB) *RatingStore {
	return &RatingStore{db: db}
}

// Upsert inserts or updates a rating.
func (s *RatingStore) Upsert(ctx context.Context, rating *Rating) error {
	query := `
		INSERT INTO movie_ratings (movie_id, rater_id, rating)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			rating = VALUES(rating),
			updated_at = CURRENT_TIMESTAMP(6)
	`
	_, err := s.db.ExecContext(ctx, query, rating.MovieID, rating.RaterID, rating.Rating)
	return err
}

// GetAggregate computes average and count for a movie's ratings.
func (s *RatingStore) GetAggregate(ctx context.Context, movieID string) (*RatingAggregate, error) {
	var agg RatingAggregate
	query := `SELECT COALESCE(ROUND(AVG(rating), 1), 0) as average, COUNT(*) as count
	          FROM movie_ratings WHERE movie_id = ?`
	err := s.db.GetContext(ctx, &agg, query, movieID)
	if err != nil {
		return nil, err
	}
	return &agg, nil
}

// Exists checks if a rating exists for a movie and rater.
func (s *RatingStore) Exists(ctx context.Context, movieID, raterID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM movie_ratings WHERE movie_id = ? AND rater_id = ?`
	err := s.db.GetContext(ctx, &count, query, movieID, raterID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
