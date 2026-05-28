package repository

import (
	"context"
	"encoding/json"

	"notification-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HistoryRepository struct {
	db *pgxpool.Pool
}

func NewHistoryRepository(db *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) CreateHistory(ctx context.Context, history *models.NotificationHistory) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO notification_history (
			event_type, channel_id, channel_name, channel_type, recipient, subject,
			message, payload, status, error_message, retry_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, sent_at;
	`,
		history.EventType,
		history.ChannelID,
		history.ChannelName,
		history.ChannelType,
		history.Recipient,
		history.Subject,
		history.Message,
		jsonOrNil(history.Payload),
		history.Status,
		history.ErrorMessage,
		history.RetryCount,
	).Scan(&history.ID, &history.CreatedAt, &history.SentAt)
}

func (r *HistoryRepository) ListHistory(ctx context.Context, filters models.HistoryFilters) ([]models.NotificationHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, event_type, channel_id, channel_name, channel_type, recipient, subject,
			message, payload, status, error_message, retry_count, created_at, sent_at
		FROM notification_history
		WHERE ($1 = '' OR status = $1)
		AND ($2 = '' OR channel_type = $2)
		AND ($3 = '' OR event_type = $3)
		ORDER BY created_at DESC, id DESC;
	`, filters.Status, filters.ChannelType, filters.EventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]models.NotificationHistory, 0)
	for rows.Next() {
		item, err := scanHistory(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, *item)
	}

	return history, rows.Err()
}

func (r *HistoryRepository) GetHistoryByID(ctx context.Context, id string) (*models.NotificationHistory, error) {
	return scanHistory(r.db.QueryRow(ctx, `
		SELECT id, event_type, channel_id, channel_name, channel_type, recipient, subject,
			message, payload, status, error_message, retry_count, created_at, sent_at
		FROM notification_history
		WHERE id = $1;
	`, id))
}

func (r *HistoryRepository) UpdateHistoryStatus(ctx context.Context, id int64, status string, errorMessage *string) error {
	var sentAtExpr string
	if status == models.NotificationStatusSent {
		sentAtExpr = "NOW()"
	} else {
		sentAtExpr = "NULL"
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE notification_history
		SET status = $2, error_message = $3, sent_at = `+sentAtExpr+`
		WHERE id = $1;
	`, id, status, errorMessage)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *HistoryRepository) IncrementRetryCount(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notification_history
		SET retry_count = retry_count + 1
		WHERE id = $1;
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *HistoryRepository) ListFailedHistory(ctx context.Context) ([]models.NotificationHistory, error) {
	return r.ListHistory(ctx, models.HistoryFilters{Status: models.NotificationStatusFailed})
}

func jsonOrNil(payload json.RawMessage) any {
	if len(payload) == 0 {
		return nil
	}

	return payload
}

func scanHistory(row rowScanner) (*models.NotificationHistory, error) {
	var history models.NotificationHistory
	if err := row.Scan(
		&history.ID,
		&history.EventType,
		&history.ChannelID,
		&history.ChannelName,
		&history.ChannelType,
		&history.Recipient,
		&history.Subject,
		&history.Message,
		&history.Payload,
		&history.Status,
		&history.ErrorMessage,
		&history.RetryCount,
		&history.CreatedAt,
		&history.SentAt,
	); err != nil {
		return nil, err
	}

	return &history, nil
}
