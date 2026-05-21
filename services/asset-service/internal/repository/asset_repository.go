package repository

import (
	"context"

	"asset-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetRepository struct {
	db *pgxpool.Pool
}

func NewAssetRepository(db *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{db: db}
}

func (r *AssetRepository) Create(ctx context.Context, asset *models.Asset) error {
	query := `
		INSERT INTO assets (id, name, type, location, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at;
	`

	return r.db.QueryRow(
		ctx,
		query,
		asset.ID,
		asset.Name,
		asset.Type,
		asset.Location,
		asset.Status,
	).Scan(&asset.CreatedAt)
}

func (r *AssetRepository) List(ctx context.Context) ([]models.Asset, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, type, location, status, created_at
		FROM assets
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]models.Asset, 0)

	for rows.Next() {
		var asset models.Asset

		if err := rows.Scan(
			&asset.ID,
			&asset.Name,
			&asset.Type,
			&asset.Location,
			&asset.Status,
			&asset.CreatedAt,
		); err != nil {
			return nil, err
		}

		assets = append(assets, asset)
	}

	return assets, rows.Err()
}

func (r *AssetRepository) GetByID(ctx context.Context, id string) (*models.Asset, error) {
	var asset models.Asset

	query := `
		SELECT id, name, type, location, status, created_at
		FROM assets
		WHERE id = $1;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&asset.ID,
		&asset.Name,
		&asset.Type,
		&asset.Location,
		&asset.Status,
		&asset.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

func (r *AssetRepository) Update(ctx context.Context, id string, asset *models.Asset) error {
	query := `
		UPDATE assets
		SET name = $1, type = $2, location = $3, status = $4
		WHERE id = $5
		RETURNING id, name, type, location, status, created_at;
	`

	return r.db.QueryRow(
		ctx,
		query,
		asset.Name,
		asset.Type,
		asset.Location,
		asset.Status,
		id,
	).Scan(
		&asset.ID,
		&asset.Name,
		&asset.Type,
		&asset.Location,
		&asset.Status,
		&asset.CreatedAt,
	)
}

func (r *AssetRepository) Delete(ctx context.Context, id string) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
