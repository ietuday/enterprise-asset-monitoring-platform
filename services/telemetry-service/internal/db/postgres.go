package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func CreateTelemetryTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS telemetry (
		id BIGSERIAL PRIMARY KEY,
		asset_id TEXT NOT NULL,
		temperature DOUBLE PRECISION NOT NULL,
		cpu DOUBLE PRECISION NOT NULL,
		memory DOUBLE PRECISION NOT NULL,
		status TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_telemetry_asset_id_created_at
	ON telemetry(asset_id, created_at DESC);
	`

	_, err := pool.Exec(ctx, query)
	return err
}
