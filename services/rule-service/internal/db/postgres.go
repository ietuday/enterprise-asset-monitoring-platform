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
		threshold DOUBLE PRECISION NOT NULL DEFAULT 0,
		value TEXT,
		severity TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT false,
		status TEXT NOT NULL DEFAULT 'draft',
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	ALTER TABLE monitoring_rules
	ADD COLUMN IF NOT EXISTS value TEXT;

	ALTER TABLE monitoring_rules
	ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'draft';

	UPDATE monitoring_rules
	SET status = CASE
		WHEN enabled = true THEN 'active'
		ELSE 'draft'
	END
	WHERE status IS NULL OR status = '';

	CREATE INDEX IF NOT EXISTS idx_monitoring_rules_status
	ON monitoring_rules(status);

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
