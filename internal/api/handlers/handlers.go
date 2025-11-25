package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/robin-camp/movies/internal/api/middleware"
	"github.com/robin-camp/movies/internal/clients/boxoffice"
	"github.com/robin-camp/movies/internal/store"
)

// MovieHandler handles movie-related endpoints.
type MovieHandler struct {
	movieStore *store.MovieStore
	boClient   *boxoffice.Client
	logger     *slog.Logger
}

// NewMovieHandler creates a MovieHandler.
func NewMovieHandler(ms *store.MovieStore, bo *boxoffice.Client, logger *slog.Logger) *MovieHandler {
	return &MovieHandler{movieStore: ms, boClient: bo, logger: logger}
}

// CreateRequest represents POST /movies body.
type CreateRequest struct {
	Title       string  `json:"title"`
	Genre       string  `json:"genre"`
	ReleaseDate string  `json:"releaseDate"`
	Distributor *string `json:"distributor,omitempty"`
	Budget      *int64  `json:"budget,omitempty"`
	MPARating   *string `json:"mpaRating,omitempty"`
}

// Create handles POST /movies.
func (h *MovieHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Genre == "" || req.ReleaseDate == "" {
		writeError(w, "BAD_REQUEST", "title, genre, and releaseDate are required", http.StatusBadRequest)
		return
	}

	releaseDate, err := time.Parse("2006-01-02", req.ReleaseDate)
	if err != nil {
		writeError(w, "BAD_REQUEST", "releaseDate must be in YYYY-MM-DD format", http.StatusBadRequest)
		return
	}

	movieID := ulid.Make().String()
	movie := &store.Movie{
		ID:          movieID,
		Title:       req.Title,
		ReleaseDate: releaseDate,
		Genre:       req.Genre,
		Distributor: req.Distributor,
		Budget:      req.Budget,
		MPARating:   req.MPARating,
	}

	if err := h.movieStore.Create(r.Context(), movie); err != nil {
		h.logger.Error("failed to create movie", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to create movie", http.StatusInternalServerError)
		return
	}

	// Enrich with box office data
	h.enrichBoxOffice(r.Context(), movie)

	// Reload box office data if present
	bo, _ := h.movieStore.GetBoxOffice(r.Context(), movieID)
	if bo != nil {
		movie.BoxOffice = &store.BoxOffice{
			Currency:    bo.Currency,
			Source:      bo.Source,
			LastUpdated: bo.LastReported,
		}
		movie.BoxOffice.Revenue.Worldwide = bo.GrossUSD
		if bo.OpeningWeekendUSA != nil {
			movie.BoxOffice.Revenue.OpeningWeekendUSA = bo.OpeningWeekendUSA
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", buildAbsoluteURL(r, "/movies/"+movie.Title))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(movie)
}

func (h *MovieHandler) enrichBoxOffice(ctx context.Context, movie *store.Movie) {
	boResp, err := h.boClient.GetByTitle(ctx, movie.Title)
	if err != nil || boResp == nil {
		h.logger.Warn("box office enrichment skipped", "title", movie.Title, "err", err)
		return
	}

	// User-provided values take precedence
	if movie.Distributor == nil && boResp.Distributor != "" {
		movie.Distributor = &boResp.Distributor
	}
	if movie.Budget == nil && boResp.Budget > 0 {
		movie.Budget = &boResp.Budget
	}
	if movie.MPARating == nil && boResp.MPARating != "" {
		movie.MPARating = &boResp.MPARating
	}

	// Store box office data
	boRow := &store.BoxOfficeRow{
		MovieID:      movie.ID,
		GrossUSD:     boResp.Revenue.Worldwide,
		Currency:     "USD",
		Source:       "ExampleBoxOfficeAPI",
		LastReported: time.Now().UTC(),
	}
	if boResp.Revenue.OpeningWeekendUSA > 0 {
		boRow.OpeningWeekendUSA = &boResp.Revenue.OpeningWeekendUSA
	}

	if err := h.movieStore.SetBoxOffice(ctx, movie.ID, boRow); err != nil {
		h.logger.Warn("failed to store box office data", "err", err)
	}
}

// List handles GET /movies.
func (h *MovieHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var year *int
	if yStr := q.Get("year"); yStr != "" {
		y, err := strconv.Atoi(yStr)
		if err != nil {
			writeError(w, "BAD_REQUEST", "Invalid year parameter", http.StatusBadRequest)
			return
		}
		year = &y
	}

	var budget *int64
	if bStr := q.Get("budget"); bStr != "" {
		b, err := strconv.ParseInt(bStr, 10, 64)
		if err != nil {
			writeError(w, "BAD_REQUEST", "Invalid budget parameter", http.StatusBadRequest)
			return
		}
		budget = &b
	}

	limit := 20
	if lStr := q.Get("limit"); lStr != "" {
		l, err := strconv.Atoi(lStr)
		if err != nil || l < 1 {
			writeError(w, "BAD_REQUEST", "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = l
	}

	var cursor *store.Cursor
	if cStr := q.Get("cursor"); cStr != "" {
		c, err := store.DecodeCursor(cStr)
		if err != nil {
			writeError(w, "BAD_REQUEST", "Invalid cursor parameter", http.StatusBadRequest)
			return
		}
		cursor = c
	}

	filters := store.ListFilters{
		Query:       q.Get("q"),
		Year:        year,
		Genre:       q.Get("genre"),
		Distributor: q.Get("distributor"),
		Budget:      budget,
		MPARating:   q.Get("mpaRating"),
		Limit:       limit,
		Cursor:      cursor,
	}

	movies, nextCursor, err := h.movieStore.List(r.Context(), filters)
	if err != nil {
		h.logger.Error("failed to list movies", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to list movies", http.StatusInternalServerError)
		return
	}

	// Enrich with box office data
	for i := range movies {
		bo, _ := h.movieStore.GetBoxOffice(r.Context(), movies[i].ID)
		if bo != nil {
			movies[i].BoxOffice = &store.BoxOffice{
				Currency:    bo.Currency,
				Source:      bo.Source,
				LastUpdated: bo.LastReported,
			}
			movies[i].BoxOffice.Revenue.Worldwide = bo.GrossUSD
			if bo.OpeningWeekendUSA != nil {
				movies[i].BoxOffice.Revenue.OpeningWeekendUSA = bo.OpeningWeekendUSA
			}
		}
	}

	resp := map[string]interface{}{"items": movies}
	if nextCursor != nil {
		encoded, _ := store.EncodeCursor(*nextCursor)
		resp["nextCursor"] = encoded
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// RatingHandler handles rating endpoints.
type RatingHandler struct {
	movieStore  *store.MovieStore
	ratingStore *store.RatingStore
	logger      *slog.Logger
}

// NewRatingHandler creates a RatingHandler.
func NewRatingHandler(ms *store.MovieStore, rs *store.RatingStore, logger *slog.Logger) *RatingHandler {
	return &RatingHandler{movieStore: ms, ratingStore: rs, logger: logger}
}

// SubmitRequest represents POST /movies/{title}/ratings body.
type SubmitRequest struct {
	Rating float64 `json:"rating"`
}

// SubmitRating handles POST /movies/{title}/ratings.
func (h *RatingHandler) SubmitRating(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	if title == "" {
		writeError(w, "BAD_REQUEST", "Missing title parameter", http.StatusBadRequest)
		return
	}
	raterID := middleware.GetRaterID(r.Context())

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "BAD_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidRating(req.Rating) {
		writeError(w, "BAD_REQUEST", "rating must be in [0.5, 1.0, ..., 5.0]", http.StatusUnprocessableEntity)
		return
	}

	movie, err := h.movieStore.GetByTitle(r.Context(), title)
	if err != nil {
		h.logger.Error("failed to get movie", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to get movie", http.StatusInternalServerError)
		return
	}
	if movie == nil {
		writeError(w, "NOT_FOUND", "Movie not found", http.StatusNotFound)
		return
	}

	exists, err := h.ratingStore.Exists(r.Context(), movie.ID, raterID)
	if err != nil {
		h.logger.Error("failed to check rating existence", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to check rating", http.StatusInternalServerError)
		return
	}

	rating := &store.Rating{
		MovieID: movie.ID,
		RaterID: raterID,
		Rating:  req.Rating,
	}

	if err := h.ratingStore.Upsert(r.Context(), rating); err != nil {
		h.logger.Error("failed to upsert rating", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to submit rating", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"movieTitle": title,
		"raterId":    raterID,
		"rating":     req.Rating,
	}

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", buildAbsoluteURL(r, fmt.Sprintf("/movies/%s/ratings", title)))
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// GetAggregate handles GET /movies/{title}/rating.
func (h *RatingHandler) GetAggregate(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	if title == "" {
		writeError(w, "BAD_REQUEST", "Missing title parameter", http.StatusBadRequest)
		return
	}

	movie, err := h.movieStore.GetByTitle(r.Context(), title)
	if err != nil {
		h.logger.Error("failed to get movie", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to get movie", http.StatusInternalServerError)
		return
	}
	if movie == nil {
		writeError(w, "NOT_FOUND", "Movie not found", http.StatusNotFound)
		return
	}

	agg, err := h.ratingStore.GetAggregate(r.Context(), movie.ID)
	if err != nil {
		h.logger.Error("failed to get aggregate", "err", err)
		writeError(w, "INTERNAL_ERROR", "Failed to get rating aggregate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agg)
}

func isValidRating(r float64) bool {
	validRatings := []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0}
	for _, v := range validRatings {
		if r == v {
			return true
		}
	}
	return false
}

// buildAbsoluteURL constructs an absolute URL from the request.
func buildAbsoluteURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

// HealthCheck handles GET /healthz.
func HealthCheck(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "unhealthy")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}
}
