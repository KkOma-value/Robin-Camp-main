package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// DB wraps sqlx.DB with convenience methods.
type DB struct {
	*sqlx.DB
	logger *slog.Logger
}

// Connect opens a MySQL connection with retries and validation.
func Connect(ctx context.Context, dsn string, logger *slog.Logger) (*DB, error) {
	var db *sqlx.DB
	var err error

	retries := 5
	for i := 0; i < retries; i++ {
		db, err = sqlx.ConnectContext(ctx, "mysql", dsn)
		if err == nil {
			break
		}
		logger.Warn("db connection attempt failed", "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second * time.Duration(i+1)):
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", retries, err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("database connected")
	return &DB{DB: db, logger: logger}, nil
}

// Ping checks database reachability for health checks.
func (d *DB) Ping(ctx context.Context) error {
	return d.PingContext(ctx)
}

// Close gracefully closes the database connection.
func (d *DB) Close() error {
	return d.DB.Close()
}

// InTx runs a function inside a transaction, rolling back on error.
func (d *DB) InTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := d.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
