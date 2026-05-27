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

func CreateNotificationTables(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS notification_channels (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		target TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS notification_history (
		id BIGSERIAL PRIMARY KEY,
		event_type TEXT NOT NULL,
		channel_id BIGINT NULL REFERENCES notification_channels(id) ON DELETE SET NULL,
		channel_name TEXT NOT NULL,
		channel_type TEXT NOT NULL,
		recipient TEXT NOT NULL,
		subject TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		payload JSONB NULL,
		status TEXT NOT NULL,
		error_message TEXT NULL,
		retry_count INT NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		sent_at TIMESTAMP NULL
	);

	CREATE INDEX IF NOT EXISTS idx_notification_channels_enabled
	ON notification_channels(enabled);

	CREATE INDEX IF NOT EXISTS idx_notification_history_status
	ON notification_history(status);

	CREATE INDEX IF NOT EXISTS idx_notification_history_channel_type
	ON notification_history(channel_type);

	CREATE INDEX IF NOT EXISTS idx_notification_history_event_type
	ON notification_history(event_type);
	`

	_, err := pool.Exec(ctx, query)
	return err
}
