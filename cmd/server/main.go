package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/robin-camp/movies/internal/config"
	"github.com/robin-camp/movies/internal/logging"
	"github.com/robin-camp/movies/internal/server"
	"github.com/robin-camp/movies/internal/store"
)

func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Connect(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Error("failed to connect to database", "err", err)
		panic(err)
	}
	defer db.Close()

	srv := server.New(cfg, db, logger)

	if err := srv.Run(ctx); err != nil {
		logger.Error("server exited with error", "err", err)
	}
}
