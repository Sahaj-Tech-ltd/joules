package db

import (
	"context"
	_ "embed"

	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

//go:embed migrations.sql
var migrationsSQL string

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	slog.Info("connected to database")

	if err := initSchema(pingCtx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users')",
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		slog.Info("initializing database schema")
		if _, err = pool.Exec(ctx, schemaSQL); err != nil {
			return err
		}
	}

	slog.Info("running migrations")
	_, err = pool.Exec(ctx, migrationsSQL)
	return err
}
