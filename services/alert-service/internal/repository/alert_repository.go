package repository

import (
	"context"

	"alert-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRepository struct {
	db *pgxpool.Pool
}

func NewAlertRepository(db *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	existingAlert, err := r.FindActiveByAssetAndName(ctx, alert.AssetID, alert.Name)
	if err == nil && existingAlert != nil {
		*alert = *existingAlert
		return nil
	}

	query := `
		INSERT INTO alerts (asset_id, name, severity, status, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at;
	`

	return r.db.QueryRow(
		ctx,
		query,
		alert.AssetID,
		alert.Name,
		alert.Severity,
		alert.Status,
		alert.Message,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)
}

func (r *AlertRepository) FindActiveByAssetAndName(ctx context.Context, assetID string, name string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		SELECT id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at
		FROM alerts
		WHERE asset_id = $1
		AND name = $2
		AND status IN ('OPEN', 'ACKNOWLEDGED')
		ORDER BY created_at DESC
		LIMIT 1;
	`

	err := r.db.QueryRow(ctx, query, assetID, name).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) List(ctx context.Context) ([]models.Alert, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at
		FROM alerts
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)

	for rows.Next() {
		var alert models.Alert

		if err := rows.Scan(
			&alert.ID,
			&alert.AssetID,
			&alert.Name,
			&alert.Severity,
			&alert.Status,
			&alert.Message,
			&alert.CreatedAt,
			&alert.UpdatedAt,
			&alert.ResolvedAt,
		); err != nil {
			return nil, err
		}

		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

func (r *AlertRepository) GetByID(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		SELECT id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at
		FROM alerts
		WHERE id = $1;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) Acknowledge(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		UPDATE alerts
		SET status = 'ACKNOWLEDGED', updated_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) Resolve(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		UPDATE alerts
		SET status = 'RESOLVED', updated_at = NOW(), resolved_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) ResolveActiveByAssetAndName(ctx context.Context, assetID string, name string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		UPDATE alerts
		SET status = 'RESOLVED', updated_at = NOW(), resolved_at = NOW()
		WHERE id = (
			SELECT id
			FROM alerts
			WHERE asset_id = $1
			AND name = $2
			AND status IN ('OPEN', 'ACKNOWLEDGED')
			ORDER BY created_at DESC
			LIMIT 1
		)
		RETURNING id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at;
	`

	err := r.db.QueryRow(ctx, query, assetID, name).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}
