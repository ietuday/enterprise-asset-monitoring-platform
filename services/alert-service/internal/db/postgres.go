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

func CreateIncidentTables(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS incidents (
		id BIGSERIAL PRIMARY KEY,
		alert_id BIGINT NULL REFERENCES alerts(id) ON DELETE SET NULL,
		asset_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		severity TEXT NOT NULL,
		status TEXT NOT NULL,
		assigned_to TEXT NULL,
		resolution_note TEXT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		acknowledged_at TIMESTAMP NULL,
		resolved_at TIMESTAMP NULL,
		closed_at TIMESTAMP NULL
	);

	CREATE TABLE IF NOT EXISTS incident_history (
		id BIGSERIAL PRIMARY KEY,
		incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
		action TEXT NOT NULL,
		old_status TEXT NULL,
		new_status TEXT NOT NULL,
		actor TEXT NOT NULL,
		comment TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_incidents_alert_id
	ON incidents(alert_id);

	CREATE INDEX IF NOT EXISTS idx_incidents_status
	ON incidents(status);

	CREATE INDEX IF NOT EXISTS idx_incidents_severity
	ON incidents(severity);

	CREATE INDEX IF NOT EXISTS idx_incidents_assigned_to
	ON incidents(assigned_to);

	CREATE INDEX IF NOT EXISTS idx_incident_history_incident_id
	ON incident_history(incident_id);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_incidents_active_alert_unique
	ON incidents(alert_id)
	WHERE alert_id IS NOT NULL AND status IN ('OPEN', 'ASSIGNED', 'ACKNOWLEDGED');
	`

	_, err := pool.Exec(ctx, query)
	return err
}
