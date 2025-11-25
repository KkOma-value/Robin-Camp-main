package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/robin-camp/movies/internal/api/handlers"
	"github.com/robin-camp/movies/internal/api/middleware"
	"github.com/robin-camp/movies/internal/clients/boxoffice"
	"github.com/robin-camp/movies/internal/config"
	"github.com/robin-camp/movies/internal/store"
)

// Server coordinates the HTTP listener and graceful shutdown lifecycle.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	db         *store.DB
}

// New wires a chi router and prepares the HTTP server instance.
func New(cfg config.Config, db *store.DB, logger *slog.Logger) *Server {
	router := chi.NewRouter()
	router.Use(middleware.Logger(logger))

	// Health check
	router.Get("/healthz", handlers.HealthCheck(db))

	// Box office client
	boClient := boxoffice.NewClient(cfg.BoxOfficeURL, cfg.BoxOfficeKey, logger)

	// Stores
	movieStore := store.NewMovieStore(db)
	ratingStore := store.NewRatingStore(db)

	// Handlers
	movieHandler := handlers.NewMovieHandler(movieStore, boClient, logger)
	ratingHandler := handlers.NewRatingHandler(movieStore, ratingStore, logger)

	// Movie routes
	router.With(middleware.BearerAuth(cfg.AuthToken)).Post("/movies", movieHandler.Create)
	router.Get("/movies", movieHandler.List)

	// Rating routes
	router.With(middleware.RequireRaterID).Post("/movies/{title}/ratings", ratingHandler.SubmitRating)
	router.Get("/movies/{title}/rating", ratingHandler.GetAggregate)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{httpServer: srv, logger: logger, db: db}
}

// Run starts the HTTP server and blocks until context cancellation or server failure.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http server starting", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.logger.Info("http server shutting down")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
