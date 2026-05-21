package repository

import (
	"context"

	"telemetry-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TelemetryRepository struct {
	db *pgxpool.Pool
}

func NewTelemetryRepository(db *pgxpool.Pool) *TelemetryRepository {
	return &TelemetryRepository{db: db}
}

func (r *TelemetryRepository) Create(ctx context.Context, telemetry *models.Telemetry) error {
	query := `
		INSERT INTO telemetry (asset_id, temperature, cpu, memory, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at;
	`

	return r.db.QueryRow(
		ctx,
		query,
		telemetry.AssetID,
		telemetry.Temperature,
		telemetry.CPU,
		telemetry.Memory,
		telemetry.Status,
	).Scan(&telemetry.ID, &telemetry.CreatedAt)
}

func (r *TelemetryRepository) GetLatestByAssetID(ctx context.Context, assetID string) (*models.Telemetry, error) {
	var telemetry models.Telemetry

	query := `
		SELECT id, asset_id, temperature, cpu, memory, status, created_at
		FROM telemetry
		WHERE asset_id = $1
		ORDER BY created_at DESC
		LIMIT 1;
	`

	err := r.db.QueryRow(ctx, query, assetID).Scan(
		&telemetry.ID,
		&telemetry.AssetID,
		&telemetry.Temperature,
		&telemetry.CPU,
		&telemetry.Memory,
		&telemetry.Status,
		&telemetry.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &telemetry, nil
}
