package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://monitoring_user:monitoring_pass@localhost:5435/monitoring_db?sslmode=disable"
	}

	return pgxpool.New(ctx, databaseURL)
}

func Init(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS monitoring_rules (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		metric TEXT NOT NULL,
		operator TEXT NOT NULL,
		threshold DOUBLE PRECISION NOT NULL,
		severity TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS rule_audit_logs (
		id BIGSERIAL PRIMARY KEY,
		rule_id BIGINT,
		action TEXT NOT NULL,
		rule_name TEXT,
		old_value JSONB,
		new_value JSONB,
		changed_by TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	_, err := pool.Exec(ctx, query)
	return err
}
