package repository

import (
	"context"
	"errors"

	"alert-service/internal/models"

	"github.com/jackc/pgx/v5"
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

func (r *AlertRepository) CreateIncident(ctx context.Context, incident *models.Incident, actor string, comment string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if incident.Status == "" {
		incident.Status = models.IncidentStatusOpen
	}

	if incident.AlertID != nil {
		existing, err := r.findActiveIncidentByAlertID(ctx, tx, *incident.AlertID)
		if err == nil {
			*incident = *existing
			return tx.Commit(ctx)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO incidents (alert_id, asset_id, title, description, severity, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (alert_id)
		WHERE alert_id IS NOT NULL AND status IN ('OPEN', 'ASSIGNED', 'ACKNOWLEDGED')
		DO NOTHING
		RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
	`, incident.AlertID, incident.AssetID, incident.Title, incident.Description, incident.Severity, incident.Status).Scan(
		&incident.ID,
		&incident.AlertID,
		&incident.AssetID,
		&incident.Title,
		&incident.Description,
		&incident.Severity,
		&incident.Status,
		&incident.AssignedTo,
		&incident.ResolutionNote,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.AcknowledgedAt,
		&incident.ResolvedAt,
		&incident.ClosedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) && incident.AlertID != nil {
		existing, findErr := r.findActiveIncidentByAlertID(ctx, tx, *incident.AlertID)
		if findErr != nil {
			return findErr
		}

		*incident = *existing
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}

	if actor == "" {
		actor = "system"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO incident_history (incident_id, action, old_status, new_status, actor, comment)
		VALUES ($1, 'CREATED', NULL, $2, $3, $4);
	`, incident.ID, incident.Status, actor, comment)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AlertRepository) findActiveIncidentByAlertID(ctx context.Context, tx pgx.Tx, alertID int64) (*models.Incident, error) {
	return scanIncident(tx.QueryRow(ctx, `
		SELECT id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at
		FROM incidents
		WHERE alert_id = $1
		AND status IN ('OPEN', 'ASSIGNED', 'ACKNOWLEDGED')
		ORDER BY created_at DESC
		LIMIT 1;
	`, alertID))
}

func (r *AlertRepository) ListIncidents(ctx context.Context, filters models.IncidentFilters) ([]models.Incident, error) {
	query := `
		SELECT id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at
		FROM incidents
		WHERE ($1 = '' OR status = $1)
		AND ($2 = '' OR severity = $2)
		AND ($3 = '' OR assigned_to = $3)
		ORDER BY created_at DESC;
	`

	rows, err := r.db.Query(ctx, query, filters.Status, filters.Severity, filters.AssignedTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}

		incidents = append(incidents, *incident)
	}

	return incidents, rows.Err()
}

func (r *AlertRepository) GetIncidentByID(ctx context.Context, id string) (*models.Incident, error) {
	return scanIncident(r.db.QueryRow(ctx, `
		SELECT id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at
		FROM incidents
		WHERE id = $1;
	`, id))
}

func (r *AlertRepository) AssignIncident(ctx context.Context, id string, assignedTo string, actor string, comment string) (*models.Incident, error) {
	return r.transitionIncident(ctx, id, models.IncidentStatusAssigned, actor, comment, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'ASSIGNED', assigned_to = $2, updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id, assignedTo))
	})
}

func (r *AlertRepository) AcknowledgeIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error) {
	return r.transitionIncident(ctx, id, models.IncidentStatusAcknowledged, actor, comment, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'ACKNOWLEDGED', acknowledged_at = COALESCE(acknowledged_at, NOW()), updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id))
	})
}

func (r *AlertRepository) ResolveIncident(ctx context.Context, id string, actor string, resolutionNote string) (*models.Incident, error) {
	return r.transitionIncident(ctx, id, models.IncidentStatusResolved, actor, resolutionNote, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'RESOLVED', resolution_note = $2, resolved_at = NOW(), updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id, resolutionNote))
	})
}

func (r *AlertRepository) CloseIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error) {
	return r.transitionIncident(ctx, id, models.IncidentStatusClosed, actor, comment, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'CLOSED', closed_at = NOW(), updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id))
	})
}

func (r *AlertRepository) GetIncidentHistory(ctx context.Context, incidentID string) ([]models.IncidentHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, incident_id, action, old_status, new_status, actor, comment, created_at
		FROM incident_history
		WHERE incident_id = $1
		ORDER BY created_at ASC, id ASC;
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]models.IncidentHistory, 0)
	for rows.Next() {
		item, err := scanIncidentHistory(rows)
		if err != nil {
			return nil, err
		}

		history = append(history, *item)
	}

	return history, rows.Err()
}

func (r *AlertRepository) AddIncidentHistory(ctx context.Context, history *models.IncidentHistory) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO incident_history (incident_id, action, old_status, new_status, actor, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at;
	`, history.IncidentID, history.Action, history.OldStatus, history.NewStatus, history.Actor, history.Comment).Scan(&history.ID, &history.CreatedAt)
}

func (r *AlertRepository) transitionIncident(
	ctx context.Context,
	id string,
	newStatus string,
	actor string,
	comment string,
	update func(context.Context, pgx.Tx, string) (*models.Incident, error),
) (*models.Incident, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM incidents WHERE id = $1;`, id).Scan(&oldStatus); err != nil {
		return nil, err
	}

	incident, err := update(ctx, tx, oldStatus)
	if err != nil {
		return nil, err
	}

	if actor == "" {
		actor = "system"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO incident_history (incident_id, action, old_status, new_status, actor, comment)
		VALUES ($1, $2, $3, $4, $5, $6);
	`, incident.ID, newStatus, oldStatus, newStatus, actor, comment)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return incident, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanIncident(row rowScanner) (*models.Incident, error) {
	var incident models.Incident

	err := row.Scan(
		&incident.ID,
		&incident.AlertID,
		&incident.AssetID,
		&incident.Title,
		&incident.Description,
		&incident.Severity,
		&incident.Status,
		&incident.AssignedTo,
		&incident.ResolutionNote,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.AcknowledgedAt,
		&incident.ResolvedAt,
		&incident.ClosedAt,
	)
	if err != nil {
		return nil, err
	}

	return &incident, nil
}

func scanIncidentHistory(row rowScanner) (*models.IncidentHistory, error) {
	var history models.IncidentHistory

	err := row.Scan(
		&history.ID,
		&history.IncidentID,
		&history.Action,
		&history.OldStatus,
		&history.NewStatus,
		&history.Actor,
		&history.Comment,
		&history.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &history, nil
}
