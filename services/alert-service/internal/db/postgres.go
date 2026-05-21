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

func CreateAlertsTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS alerts (
		id BIGSERIAL PRIMARY KEY,
		asset_id TEXT NOT NULL,
		name TEXT NOT NULL,
		severity TEXT NOT NULL,
		status TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		resolved_at TIMESTAMP NULL
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_asset_id
	ON alerts(asset_id);

	CREATE INDEX IF NOT EXISTS idx_alerts_status
	ON alerts(status);
	`

	_, err := pool.Exec(ctx, query)
	return err
}
