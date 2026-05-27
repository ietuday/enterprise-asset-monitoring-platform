package repository

import (
	"context"

	"notification-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChannelRepository struct {
	db *pgxpool.Pool
}

func NewChannelRepository(db *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) CreateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO notification_channels (name, type, target, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at;
	`, channel.Name, channel.Type, channel.Target, channel.Enabled).Scan(
		&channel.ID,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)
}

func (r *ChannelRepository) ListChannels(ctx context.Context) ([]models.NotificationChannel, error) {
	return r.list(ctx, `
		SELECT id, name, type, target, enabled, created_at, updated_at
		FROM notification_channels
		ORDER BY id DESC;
	`)
}

func (r *ChannelRepository) ListEnabledChannels(ctx context.Context) ([]models.NotificationChannel, error) {
	return r.list(ctx, `
		SELECT id, name, type, target, enabled, created_at, updated_at
		FROM notification_channels
		WHERE enabled = TRUE
		ORDER BY id ASC;
	`)
}

func (r *ChannelRepository) GetChannelByID(ctx context.Context, id string) (*models.NotificationChannel, error) {
	return scanChannel(r.db.QueryRow(ctx, `
		SELECT id, name, type, target, enabled, created_at, updated_at
		FROM notification_channels
		WHERE id = $1;
	`, id))
}

func (r *ChannelRepository) UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	return r.db.QueryRow(ctx, `
		UPDATE notification_channels
		SET name = $2, type = $3, target = $4, enabled = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at;
	`, channel.ID, channel.Name, channel.Type, channel.Target, channel.Enabled).Scan(
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)
}

func (r *ChannelRepository) DeleteChannel(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM notification_channels WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *ChannelRepository) SetChannelEnabled(ctx context.Context, id string, enabled bool) (*models.NotificationChannel, error) {
	return scanChannel(r.db.QueryRow(ctx, `
		UPDATE notification_channels
		SET enabled = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, type, target, enabled, created_at, updated_at;
	`, id, enabled))
}

func (r *ChannelRepository) list(ctx context.Context, query string) ([]models.NotificationChannel, error) {
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := make([]models.NotificationChannel, 0)
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, *channel)
	}

	return channels, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanChannel(row rowScanner) (*models.NotificationChannel, error) {
	var channel models.NotificationChannel
	if err := row.Scan(
		&channel.ID,
		&channel.Name,
		&channel.Type,
		&channel.Target,
		&channel.Enabled,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &channel, nil
}
