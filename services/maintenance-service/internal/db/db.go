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

func CreateMaintenanceTables(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS maintenance_tasks (
		id BIGSERIAL PRIMARY KEY,
		asset_id TEXT NOT NULL,
		title VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		maintenance_type VARCHAR(100) NOT NULL,
		priority VARCHAR(50) NOT NULL DEFAULT 'medium',
		status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
		scheduled_date TIMESTAMP NOT NULL,
		due_date TIMESTAMP NOT NULL,
		completed_at TIMESTAMP NULL,
		assigned_to VARCHAR(255) NOT NULL DEFAULT '',
		created_by VARCHAR(255) NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS maintenance_history (
		id BIGSERIAL PRIMARY KEY,
		task_id BIGINT NOT NULL REFERENCES maintenance_tasks(id) ON DELETE CASCADE,
		action VARCHAR(100) NOT NULL,
		old_status VARCHAR(50) NOT NULL DEFAULT '',
		new_status VARCHAR(50) NOT NULL DEFAULT '',
		comment TEXT NOT NULL DEFAULT '',
		performed_by VARCHAR(255) NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	ALTER TABLE maintenance_tasks
	ALTER COLUMN asset_id TYPE TEXT
	USING asset_id::TEXT;

	CREATE INDEX IF NOT EXISTS idx_maintenance_tasks_asset_id
	ON maintenance_tasks(asset_id);

	CREATE INDEX IF NOT EXISTS idx_maintenance_tasks_status
	ON maintenance_tasks(status);

	CREATE INDEX IF NOT EXISTS idx_maintenance_tasks_priority
	ON maintenance_tasks(priority);

	CREATE INDEX IF NOT EXISTS idx_maintenance_tasks_due_date
	ON maintenance_tasks(due_date);

	CREATE INDEX IF NOT EXISTS idx_maintenance_history_task_id
	ON maintenance_history(task_id);
	`

	_, err := pool.Exec(ctx, query)
	return err
}
